并发调度
-------------

垃圾回收、系统监控、网络通信、文件读写、还有用户并发任务等等，所有这些都需要一个高效且聪明的调度器来只会协调。语言内置运行时，在进程和线程的基础上做更高层次的抽象是现代语言最流行的的做法。

Go 通过全新的架构模型，刻意地模糊线程和协程的概念，通过三种基本都UI想相互协作，来实现在用户控件管理和调度并发任务：

* Processor（P）：作用类似于 CPU 核，用来控制可同时并发执行的任务数量。每个工作线程都必须绑定一个有效的 P 才被允许执行任务，否则只能休眠。P 还为线程提供资源，比如对象分配内存、本地任务队列等。线程独享所绑定的 P 资源，可在无锁状态下执行高效操作。
* Goroutine（G）：进程内的一切都在以 Goroutine 方式运行，包括运行时相关服务。G 并非执行体，它仅仅保存并发任务的状态，为任务执行提供所需的栈空间。G 任务创建之后被放置在 P 的本地队列或全局队列，等待工作线程调度执行。
* Machine（M）：实际的执行体是系统线程（M），它和 P 绑定，以调度循环的方式不停执行 G 并发任务。M 通过修改寄存器，将执行栈指向 G 自带的栈内存，并在此空间分配栈帧，执行任务函数。

通常情况下，P 的数量相对恒定，默认与 CPU 核数量相同，但也可能更多或更少。而 M 则是由调度器按需创建的。



### 初始化

调度器的初始化函数 `schedinit` 除了内存分配、垃圾回收等操作外，针对自身的初始化无非是 `maxMcount`、`GOMAXPROCS` 等。

```go
//
// runtime/proc.go
//

// The bootstrap sequence is:
//
//	call osinit
//	call schedinit
//	make & queue new G
//	call runtime·mstart
//
// The new G calls runtime·main.
func schedinit() {
	sched.maxmcount = 10000

	// The world starts stopped.
	worldStopped()

	goargs()
	goenvs()
	parsedebugvars()
	gcinit()

	lock(&sched.lock)
	sched.lastpoll = uint64(nanotime())
	procs := ncpu
	if n, ok := atoi32(gogetenv("GOMAXPROCS")); ok && n > 0 {
		procs = n
	}
	if procresize(procs) != nil {
		throw("unknown runnable goroutine during bootstrap")
	}
	unlock(&sched.lock)

	// World is effectively started now, as P's can run.
	worldStarted()
}

// Change number of processors.
//
// sched.lock must be held, and the world must be stopped.
//
// gcworkbufs must not be being modified by either the GC or the write barrier
// code, so the GC must not be running if the number of Ps actually changes.
//
// Returns list of Ps with local work, they need to be scheduled by the caller.
func procresize(nprocs int32) *p {
	assertLockHeld(&sched.lock)
	assertWorldStopped()

	old := gomaxprocs
	if old < 0 || nprocs <= 0 {
		throw("procresize: invalid arg")
	}
	if trace.enabled {
		traceGomaxprocs(nprocs)
	}

	// update statistics
	now := nanotime()
	if sched.procresizetime != 0 {
		sched.totaltime += int64(old) * (now - sched.procresizetime)
	}
	sched.procresizetime = now

	maskWords := (nprocs + 31) / 32

	// Grow allp if necessary.
	if nprocs > int32(len(allp)) {
		// Synchronize with retake, which could be running
		// concurrently since it doesn't run on a P.
		lock(&allpLock)
		if nprocs <= int32(cap(allp)) {
			allp = allp[:nprocs]
		} else {
			nallp := make([]*p, nprocs)
			// Copy everything up to allp's cap so we
			// never lose old allocated Ps.
			copy(nallp, allp[:cap(allp)])
			allp = nallp
		}

		if maskWords <= int32(cap(idlepMask)) {
			idlepMask = idlepMask[:maskWords]
			timerpMask = timerpMask[:maskWords]
		} else {
			nidlepMask := make([]uint32, maskWords)
			// No need to copy beyond len, old Ps are irrelevant.
			copy(nidlepMask, idlepMask)
			idlepMask = nidlepMask

			ntimerpMask := make([]uint32, maskWords)
			copy(ntimerpMask, timerpMask)
			timerpMask = ntimerpMask
		}
		unlock(&allpLock)
	}

	// initialize new P's
	for i := old; i < nprocs; i++ {
		pp := allp[i]
		if pp == nil {
			pp = new(p)
		}
		pp.init(i)
		atomicstorep(unsafe.Pointer(&allp[i]), unsafe.Pointer(pp))
	}

	_g_ := getg()
	if _g_.m.p != 0 && _g_.m.p.ptr().id < nprocs {
		// continue to use the current P
		_g_.m.p.ptr().status = _Prunning
		_g_.m.p.ptr().mcache.prepareForSweep()
	} else {
		// release the current P and acquire allp[0].
		//
		// We must do this before destroying our current P
		// because p.destroy itself has write barriers, so we
		// need to do that from a valid P.
		if _g_.m.p != 0 {
			if trace.enabled {
				// Pretend that we were descheduled
				// and then scheduled again to keep
				// the trace sane.
				traceGoSched()
				traceProcStop(_g_.m.p.ptr())
			}
			_g_.m.p.ptr().m = 0
		}
		_g_.m.p = 0
		p := allp[0]
		p.m = 0
		p.status = _Pidle
		acquirep(p)
		if trace.enabled {
			traceGoStart()
		}
	}

	// g.m.p is now set, so we no longer need mcache0 for bootstrapping.
	mcache0 = nil

	// release resources from unused P's
	for i := nprocs; i < old; i++ {
		p := allp[i]
		p.destroy()
		// can't free P itself because it can be referenced by an M in syscall
	}

	// Trim allp.
	if int32(len(allp)) != nprocs {
		lock(&allpLock)
		allp = allp[:nprocs]
		idlepMask = idlepMask[:maskWords]
		timerpMask = timerpMask[:maskWords]
		unlock(&allpLock)
	}

	var runnablePs *p
	for i := nprocs - 1; i >= 0; i-- {
		p := allp[i]
		if _g_.m.p.ptr() == p {
			continue
		}
		p.status = _Pidle
		if runqempty(p) {
			pidleput(p)
		} else {
			p.m.set(mget())
			p.link.set(runnablePs)
			runnablePs = p
		}
	}
	stealOrder.reset(uint32(nprocs))
	var int32p *int32 = &gomaxprocs // make compiler check that gomaxprocs is an int32
	atomic.Store((*uint32)(unsafe.Pointer(int32p)), uint32(nprocs))
	return runnablePs
}
```

在调度器初始化阶段，所有的 P 对象都是新建的。而 `startTheWorld` 会激活全部由本地任务的 P 对象。



### 任务

 编译器会将 `go func(...)`语句翻译成 `runtime.newproc` 调用，如下代码：

```go
//
// main.go
//

//go:noinline
func Add(a, b int) int {
	return a + b
}

func main() {
	a := 1
	b := 2
	go Add(a, b)
}
```

我们使用 `go tool compile -S -N -l main.go > main.S` 会得到如下反汇编内容：

```asm
"".Add STEXT nosplit size=25 args=0x18 locals=0x0 funcid=0x0
	0x0000 00000 (main.go:4)	TEXT	"".Add(SB), NOSPLIT|ABIInternal, $0-24
	0x0000 00000 (main.go:4)	MOVQ	$0, "".~r2+24(SP)
	0x0009 00009 (main.go:5)	MOVQ	"".a+8(SP), AX
	0x000e 00014 (main.go:5)	ADDQ	"".b+16(SP), AX
	0x0013 00019 (main.go:5)	MOVQ	AX, "".~r2+24(SP)
	0x0018 00024 (main.go:5)	RET

"".main STEXT size=107 args=0x0 locals=0x40 funcid=0x0
	0x0000 00000 (main.go:8)	TEXT	"".main(SB), ABIInternal, $64-0
	0x0000 00000 (main.go:8)	MOVQ	(TLS), CX
	0x0009 00009 (main.go:8)	CMPQ	SP, 16(CX)
	0x000d 00013 (main.go:8)	JLS	100
	
	0x000f 00015 (main.go:8)	SUBQ	$64, SP
	0x0013 00019 (main.go:8)	MOVQ	BP, 56(SP)
	0x0018 00024 (main.go:8)	LEAQ	56(SP), BP
	
	0x001d 00029 (main.go:9)	MOVQ	$1, "".a+48(SP) // a = 1
	0x0026 00038 (main.go:10)	MOVQ	$2, "".b+40(SP) // b = 2
	0x002f 00047 (main.go:11)	MOVQ	"".a+48(SP), AX // AX = a
	0x0034 00052 (main.go:11)	MOVL	$24, (SP)		// newproc.siz = 24
	0x003b 00059 (main.go:11)	LEAQ	"".Add·f(SB), CX // CX = Add
	0x0042 00066 (main.go:11)	MOVQ	CX, 8(SP)		// fn.fn = CX = Add
	0x0047 00071 (main.go:11)	MOVQ	AX, 16(SP)		// AX = a = 1
	0x004c 00076 (main.go:11)	MOVQ	$2, 24(SP)		// 2 => b
	0x0055 00085 (main.go:11)	CALL	runtime.newproc(SB)
	0x005a 00090 (main.go:12)	MOVQ	56(SP), BP
	0x005f 00095 (main.go:12)	ADDQ	$64, SP
	0x0063 00099 (main.go:12)	RET
	0x0064 00100 (main.go:12)	NOP
	0x0064 00100 (main.go:8)	PCDATA	$1, $-1
	0x0064 00100 (main.go:8)	PCDATA	$0, $-2
	0x0064 00100 (main.go:8)	CALL	runtime.morestack_noctxt(SB)
	0x0069 00105 (main.go:8)	PCDATA	$0, $-1
	0x0069 00105 (main.go:8)	JMP	0

```

接着来看 `runtime.newproc` 的代码：

```go
//
// runtime/proc.go
//

// Create a new g running fn with siz bytes of arguments.
// Put it on the queue of g's waiting to run.
// The compiler turns a go statement into a call to this.
//
// The stack layout of this call is unusual: it assumes that the
// arguments to pass to fn are on the stack sequentially immediately
// after &fn. Hence, they are logically part of newproc's argument
// frame, even though they don't appear in its signature (and can't
// because their types differ between call sites).
//
// This must be nosplit because this stack layout means there are
// untyped arguments in newproc's argument frame. Stack copies won't
// be able to adjust them and stack splits won't be able to copy them.
//
//go:nosplit
func newproc(siz int32, fn *funcval) {
	argp := add(unsafe.Pointer(&fn), sys.PtrSize)
	gp := getg()
	pc := getcallerpc()
	systemstack(func() {
		newg := newproc1(fn, argp, siz, gp, pc)

		_p_ := getg().m.p.ptr()
		runqput(_p_, newg, true)

		if mainStarted {
			wakep()
		}
	})
}

type funcval struct {
	fn uintptr
	// variable-size, fn-specific data here
}
```

根据汇编代码所示，实际上压入了4个参数，所以实际的调用参数如下：

```go
func main() {
    runtime.newproc(24, &funcval{
        fn: Addr,
        a = 1,
        b = 2
    })
}
```

有了参数，就下去就是从栈中获取 `pc` 然后创建对应的 G 结构体。在创建过程中 G 对象默认会复用，除 P 本地的复用链表外，还有全局链表在多个 P 之间共享。在获取到 G 对象之后，`newproc1` 会进行一系列的初始化操作（不管是新建的还是复用的），同时相关执行参数也会被拷贝到 G 的栈空间。创建完成的 G 会被邮箱放入 P 本地队列等待执行（无锁操作）。



### 线程

当 `newproc1` 成功创建 G 之后，会尝试用 `wakep` 唤醒 M 执行任务。M 最特别的是自带一个名为 g0 栈大小为 8K 的一个 G 对象，这是为了在暂停用户 G 时，如果不更改执行栈，可能会造成多个线程共享内存，从而引发混乱。同时在执行垃圾回收时，也需要根据这个来收缩被线程持有的 G 栈空间。因此，当需要执行管理指令时，会将线程栈临时切换到 g0 上，与用户逻辑彻底隔离（这也就是源码中经常见到的 `systemstack` 方法所做的，它会切换到 g0 后再执行运行时的管理工作）。

```asm
// func systemstack(fn func())
TEXT runtime·systemstack(SB), NOSPLIT, $0-8
	MOVQ	fn+0(FP), DI	// DI = fn
	get_tls(CX)
	MOVQ	g(CX), AX	// AX = g
	MOVQ	g_m(AX), BX	// BX = m

	CMPQ	AX, m_gsignal(BX)
	JEQ	noswitch

	MOVQ	m_g0(BX), DX	// DX = g0
	CMPQ	AX, DX
	JEQ	noswitch

	CMPQ	AX, m_curg(BX)
	JNE	bad

	// switch stacks
	// save our state in g->sched. Pretend to
	// be systemstack_switch if the G stack is scanned.
	CALL	gosave_systemstack_switch<>(SB)

	// switch to g0
	MOVQ	DX, g(CX)
	MOVQ	DX, R14 // set the g register
	MOVQ	(g_sched+gobuf_sp)(DX), BX
	MOVQ	BX, SP

	// call target function
	MOVQ	DI, DX
	MOVQ	0(DI), DI
	CALL	DI

	// switch back to g
	get_tls(CX)
	MOVQ	g(CX), AX
	MOVQ	g_m(AX), BX
	MOVQ	m_curg(BX), AX
	MOVQ	AX, g(CX)
	MOVQ	(g_sched+gobuf_sp)(AX), SP
	MOVQ	$0, (g_sched+gobuf_sp)(AX)
	RET
```



### 执行

M 执行 G 并发任务有两个起点：线程启动函数 `mstart` 和 `stopm` 休眠唤醒后再度恢复调度循环。准备进入工作状态的 M 必须绑定一个有效 P（为 M 提供 cache，以便于为工作线程提供对象内存分配）。

当一切就绪之后，M 进入核心调度循环，一个由 `schedule`、`execute`、`goroutine fn`、`goexit` 函数组成的逻辑循环。就算 M 在休眠唤醒后，也只是从断点处恢复。



### 连续栈

连续栈将调用堆栈（call stack）所有栈帧分配在一个连续内存空间。但空间不足时，另分配 2x 内存块，并拷贝当前栈全部数据，以避免分段栈（Segmented Stack）链表结构在函数调用频繁时可能引发的切分热点（split hot）问题。

```go
//
// runtime/runtime2.go
//

type g struct {
	// Stack parameters.
	// stack describes the actual stack memory: [stack.lo, stack.hi).
	// stackguard0 is the stack pointer compared in the Go stack growth prologue.
	// It is stack.lo+StackGuard normally, but can be StackPreempt to trigger a preemption.
	// stackguard1 is the stack pointer compared in the C stack growth prologue.
	// It is stack.lo+StackGuard on g0 and gsignal stacks.
	// It is ~0 on other goroutine stacks, to trigger a call to morestackc (and crash).
	stack       stack   // offset known to runtime/cgo
	stackguard0 uintptr // offset known to liblink
	stackguard1 uintptr // offset known to liblink

	_panic       *_panic // innermost panic - offset known to liblink
	_defer       *_defer // innermost defer
	m            *m      // current m; offset known to arm liblink
	sched        gobuf
	syscallsp    uintptr        // if status==Gsyscall, syscallsp = sched.sp to use during gc
	syscallpc    uintptr        // if status==Gsyscall, syscallpc = sched.pc to use during gc
	stktopsp     uintptr        // expected sp at top of stack, to check in traceback
	param        unsafe.Pointer // passed parameter on wakeup
	atomicstatus uint32
	stackLock    uint32 // sigprof/scang lock; TODO: fold in to atomicstatus
	goid         int64
	schedlink    guintptr
	waitsince    int64      // approx time when the g become blocked
	waitreason   waitReason // if status==Gwaiting

	preempt       bool // preemption signal, duplicates stackguard0 = stackpreempt
	preemptStop   bool // transition to _Gpreempted on preemption; otherwise, just deschedule
	preemptShrink bool // shrink stack at synchronous safe point

	// asyncSafePoint is set if g is stopped at an asynchronous
	// safe point. This means there are frames on the stack
	// without precise pointer information.
	asyncSafePoint bool

	paniconfault bool // panic (instead of crash) on unexpected fault address
	gcscandone   bool // g has scanned stack; protected by _Gscan bit in status
	throwsplit   bool // must not split stack
	// activeStackChans indicates that there are unlocked channels
	// pointing into this goroutine's stack. If true, stack
	// copying needs to acquire channel locks to protect these
	// areas of the stack.
	activeStackChans bool
	// parkingOnChan indicates that the goroutine is about to
	// park on a chansend or chanrecv. Used to signal an unsafe point
	// for stack shrinking. It's a boolean value, but is updated atomically.
	parkingOnChan uint8

	raceignore     int8     // ignore race detection events
	sysblocktraced bool     // StartTrace has emitted EvGoInSyscall about this goroutine
	sysexitticks   int64    // cputicks when syscall has returned (for tracing)
	traceseq       uint64   // trace event sequencer
	tracelastp     puintptr // last P emitted an event for this goroutine
	lockedm        muintptr
	sig            uint32
	writebuf       []byte
	sigcode0       uintptr
	sigcode1       uintptr
	sigpc          uintptr
	gopc           uintptr         // pc of go statement that created this goroutine
	ancestors      *[]ancestorInfo // ancestor information goroutine(s) that created this goroutine (only used if debug.tracebackancestors)
	startpc        uintptr         // pc of goroutine function
	racectx        uintptr
	waiting        *sudog         // sudog structures this g is waiting on (that have a valid elem ptr); in lock order
	cgoCtxt        []uintptr      // cgo traceback context
	labels         unsafe.Pointer // profiler labels
	timer          *timer         // cached timer for time.Sleep
	selectDone     uint32         // are we participating in a select and did someone win the race?

	// Per-G GC state

	// gcAssistBytes is this G's GC assist credit in terms of
	// bytes allocated. If this is positive, then the G has credit
	// to allocate gcAssistBytes bytes without assisting. If this
	// is negative, then the G must correct this by performing
	// scan work. We track this in bytes to make it fast to update
	// and check for debt in the malloc hot path. The assist ratio
	// determines how this corresponds to scan work debt.
	gcAssistBytes int64
}
```

其中 `stackguard0` 是个非常重要的指针，在函数头部，编译器会插入一段指令将其与SP寄存器进行比较，从而决定是否需要对栈控件进行扩容。另外，它还被用作抢占调度的标志。栈的初始化分配发生在 `newproc1` 创建新 G 对象时，在获取栈空间后，会立即设置 stackguard0 指针。对于如下 Go 代码

```go
package main

func test() {
  println("hello world")
}
```

经过反编译之后，我们可以得到如下汇编代码：

```asm
"".test STEXT size=86 args=0x0 locals=0x18 funcid=0x0
	0x0000 00000 (main.go:3)	TEXT	"".test(SB), ABIInternal, $24-0
	0x0000 00000 (main.go:3)	MOVQ	(TLS), CX
	0x0009 00009 (main.go:3)	CMPQ	SP, 16(CX)
	0x000d 00013 (main.go:3)	JLS	79

	0x000f 00015 (main.go:3)	SUBQ	$24, SP
	0x0013 00019 (main.go:3)	MOVQ	BP, 16(SP)
	0x0018 00024 (main.go:3)	LEAQ	16(SP), BP
	0x001d 00029 (main.go:4)	NOP
	0x0020 00032 (main.go:4)	CALL	runtime.printlock(SB)
	0x0025 00037 (main.go:4)	LEAQ	go.string."hello world\n"(SB), AX
	0x002c 00044 (main.go:4)	MOVQ	AX, (SP)
	0x0030 00048 (main.go:4)	MOVQ	$12, 8(SP)
	0x0039 00057 (main.go:4)	CALL	runtime.printstring(SB)
	0x003e 00062 (main.go:4)	NOP
	0x0040 00064 (main.go:4)	CALL	runtime.printunlock(SB)
	0x0045 00069 (main.go:5)	MOVQ	16(SP), BP
	0x004a 00074 (main.go:5)	ADDQ	$24, SP
	0x004e 00078 (main.go:5)	RET

	0x004f 00079 (main.go:5)	NOP
	0x004f 00079 (main.go:3)	CALL	runtime.morestack_noctxt(SB)
	0x0054 00084 (main.go:3)	JMP	0
```

可以看到在函数头部会插入一段指令 `CMPQ	SP, 16(CX)` 表示将 stackguard0 与 SP 寄存器进行对比，如果 `sp <= stackguard0` （栈是从高地址往低地址增长）则表示当前栈已经溢出，只有在扩容栈之后，当前和后续函数才可以继续分配栈帧内存。

```plain
lo                                                              hi
+─────────────+──────────────+──────────────────────────────────+
| StackGuard  | stackguard0  |                             .... |
+─────────────+──────────────+──────────────────────────────────+
                                            <---------------- SP
```



### 系统调用

为支持并发调度，Go 专门对 syscall、cgo 进行了封装，以便在长时间阻塞时能切换执行其他任务。在 syscall 包中，将系统调用分为 Syscall 和 RawSyscall 两类。最大的不同在于 Syscall 增加了 `entrysyscall/exitsyscall` 调用，这就是允许调度的关键所在

```asm
#
# syscall/asm_linux_amd64.s
#

TEXT ·Syscall(SB),NOSPLIT,$0-56
	CALL	runtime·entersyscall(SB)
	MOVQ	a1+8(FP), DI
	MOVQ	a2+16(FP), SI
	MOVQ	a3+24(FP), DX
	MOVQ	trap+0(FP), AX	// syscall entry
	SYSCALL
	CMPQ	AX, $0xfffffffffffff001
	JLS	ok
	MOVQ	$-1, r1+32(FP)
	MOVQ	$0, r2+40(FP)
	NEGQ	AX
	MOVQ	AX, err+48(FP)
	CALL	runtime·exitsyscall(SB)
	RET
ok:
	MOVQ	AX, r1+32(FP)
	MOVQ	DX, r2+40(FP)
	MOVQ	$0, err+48(FP)
	CALL	runtime·exitsyscall(SB)
	RET
```

监控线程 sysmon 对 syscall 非常重要，因为它负责将因系统调用而长时间阻塞的 P 抢回，用于执行其他任务。

```go
//
// runtime/proc.go
//

// The goroutine g is about to enter a system call.
// Record that it's not using the cpu anymore.
// This is called only from the go syscall library and cgocall,
// not from the low-level system calls used by the runtime.
//
// Entersyscall cannot split the stack: the gosave must
// make g->sched refer to the caller's stack segment, because
// entersyscall is going to return immediately after.
//
// Nothing entersyscall calls can split the stack either.
// We cannot safely move the stack during an active call to syscall,
// because we do not know which of the uintptr arguments are
// really pointers (back into the stack).
// In practice, this means that we make the fast path run through
// entersyscall doing no-split things, and the slow path has to use systemstack
// to run bigger things on the system stack.
//
// reentersyscall is the entry point used by cgo callbacks, where explicitly
// saved SP and PC are restored. This is needed when exitsyscall will be called
// from a function further up in the call stack than the parent, as g->syscallsp
// must always point to a valid stack frame. entersyscall below is the normal
// entry point for syscalls, which obtains the SP and PC from the caller.
//
// Syscall tracing:
// At the start of a syscall we emit traceGoSysCall to capture the stack trace.
// If the syscall does not block, that is it, we do not emit any other events.
// If the syscall blocks (that is, P is retaken), retaker emits traceGoSysBlock;
// when syscall returns we emit traceGoSysExit and when the goroutine starts running
// (potentially instantly, if exitsyscallfast returns true) we emit traceGoStart.
// To ensure that traceGoSysExit is emitted strictly after traceGoSysBlock,
// we remember current value of syscalltick in m (_g_.m.syscalltick = _g_.m.p.ptr().syscalltick),
// whoever emits traceGoSysBlock increments p.syscalltick afterwards;
// and we wait for the increment before emitting traceGoSysExit.
// Note that the increment is done even if tracing is not enabled,
// because tracing can be enabled in the middle of syscall. We don't want the wait to hang.
//
//go:nosplit
func reentersyscall(pc, sp uintptr) {
	_g_ := getg()

	// Disable preemption because during this function g is in Gsyscall status,
	// but can have inconsistent g->sched, do not let GC observe it.
	_g_.m.locks++

	// Entersyscall must not call any function that might split/grow the stack.
	// (See details in comment above.)
	// Catch calls that might, by replacing the stack guard with something that
	// will trip any stack check and leaving a flag to tell newstack to die.
	_g_.stackguard0 = stackPreempt
	_g_.throwsplit = true

	// Leave SP around for GC and traceback.
	save(pc, sp)
	_g_.syscallsp = sp
	_g_.syscallpc = pc
	casgstatus(_g_, _Grunning, _Gsyscall)
	if _g_.syscallsp < _g_.stack.lo || _g_.stack.hi < _g_.syscallsp {
		systemstack(func() {
			print("entersyscall inconsistent ", hex(_g_.syscallsp), " [", hex(_g_.stack.lo), ",", hex(_g_.stack.hi), "]\n")
			throw("entersyscall")
		})
	}

	if trace.enabled {
		systemstack(traceGoSysCall)
		// systemstack itself clobbers g.sched.{pc,sp} and we might
		// need them later when the G is genuinely blocked in a
		// syscall
		save(pc, sp)
	}

	if atomic.Load(&sched.sysmonwait) != 0 {
		systemstack(entersyscall_sysmon)
		save(pc, sp)
	}

	if _g_.m.p.ptr().runSafePointFn != 0 {
		// runSafePointFn may stack split if run on this stack
		systemstack(runSafePointFn)
		save(pc, sp)
	}

	_g_.m.syscalltick = _g_.m.p.ptr().syscalltick
	_g_.sysblocktraced = true
	_g_.m.mcache = nil
	pp := _g_.m.p.ptr()
	pp.m = 0
	_g_.m.oldp.set(pp)
	_g_.m.p = 0
	atomic.Store(&pp.status, _Psyscall)
	if sched.gcwaiting != 0 {
		systemstack(entersyscall_gcwait)
		save(pc, sp)
	}

	_g_.m.locks--
}

// Standard syscall entry used by the go syscall library and normal cgo calls.
//
// This is exported via linkname to assembly in the syscall package.
//
//go:nosplit
//go:linkname entersyscall
func entersyscall() {
	reentersyscall(getcallerpc(), getcallersp())
}

func entersyscall_sysmon() {
	lock(&sched.lock)
	if atomic.Load(&sched.sysmonwait) != 0 {
		atomic.Store(&sched.sysmonwait, 0)
		notewakeup(&sched.sysmonnote)
	}
	unlock(&sched.lock)
}
```



### 监控

系统监控现场 sysmon 主要做以下几件事：

* 释放闲置时间超过 5 分钟的 span 物理内存
* 如果超过2分钟没有垃圾回收，则强制执行
* 将长时间未处理的 netpoll 结果添加到任务队列
* 向长时间运行的 G 任务发出抢占调度
* 收回因 syscall 而长时间阻塞的 P



### 其他

#### runtime.Gosched

用户可调用 `runtime.Gosched` 将当前 G 任务暂停，将其重新放回到全局队列中，让出当前 M 去执行其他任务。

#### gopark & goready

gopark 与 Gosched 最大的区别在于，gopark不会将 G 放回到待运行队列中，必须主动恢复，否则该任务会遗失。与之配套的是 goready，该方法会将 G 放回优先级最高的 P.runnext

#### Goexit

用户可调用 `runtime.Goexit` 立即终止 G 任务，不管当前处于调用堆栈的哪个层次。在终止前，它确保所有的 G.defer 被执行

>而通过执行 `os.Exit(1)` 则不会执行 defer 语句

#### stopTheWorld

用户逻辑必须暂停在一个安全点上，否则会引发很多意外问题。因此，stopTheWorld 同样是通过“通知”机制，让 G 主动停止（比如设置 gcwaiting = 1 让调度器函数 schedule 主动休眠 M；或者向所有正在运行的 G 任务发出抢占调度，使其停止）
