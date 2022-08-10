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

