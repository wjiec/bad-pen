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
"".Add STEXT nosplit size=56 args=0x10 locals=0x10 funcid=0x0
	0x0000 00000 (main.go:4)	TEXT	"".Add(SB), NOSPLIT|ABIInternal, $16-16
	0x0000 00000 (main.go:4)	SUBQ	$16, SP
	0x0004 00004 (main.go:4)	MOVQ	BP, 8(SP)
	0x0009 00009 (main.go:4)	LEAQ	8(SP), BP
	0x000e 00014 (main.go:4)	MOVQ	AX, "".a+24(SP)
	0x0013 00019 (main.go:4)	MOVQ	BX, "".b+32(SP)
	0x0018 00024 (main.go:4)	MOVQ	$0, "".~r2(SP)
	0x0020 00032 (main.go:5)	MOVQ	"".a+24(SP), AX
	0x0025 00037 (main.go:5)	ADDQ	"".b+32(SP), AX
	0x002a 00042 (main.go:5)	MOVQ	AX, "".~r2(SP)
	0x002e 00046 (main.go:5)	MOVQ	8(SP), BP
	0x0033 00051 (main.go:5)	ADDQ	$16, SP
	0x0037 00055 (main.go:5)	RET


"".main STEXT size=153 args=0x0 locals=0x40 funcid=0x0
	0x0000 00000 (main.go:8)	TEXT	"".main(SB), ABIInternal, $64-0
	0x0000 00000 (main.go:8)	CMPQ	SP, 16(R14)
	0x0004 00004 (main.go:8)	JLS	143
	0x000a 00010 (main.go:8)	SUBQ	$64, SP
	0x000e 00014 (main.go:8)	MOVQ	BP, 56(SP)
	0x0013 00019 (main.go:8)	LEAQ	56(SP), BP
	0x0018 00024 (main.go:9)	MOVQ	$1, "".a+24(SP)
	0x0021 00033 (main.go:10)	MOVQ	$2, "".b+16(SP)
	0x002a 00042 (main.go:11)	MOVQ	"".a+24(SP), CX
	0x002f 00047 (main.go:11)	MOVQ	CX, ""..autotmp_2+40(SP)
	0x0034 00052 (main.go:11)	MOVQ	"".b+16(SP), CX
	0x0039 00057 (main.go:11)	MOVQ	CX, ""..autotmp_3+32(SP)
	0x003e 00062 (main.go:11)	LEAQ	type.noalg.struct { F uintptr; ""..autotmp_2 int; ""..autotmp_3 int }(SB), AX
	0x0045 00069 (main.go:11)	CALL	runtime.newobject(SB)
	0x004a 00074 (main.go:11)	MOVQ	AX, ""..autotmp_4+48(SP)
	0x004f 00079 (main.go:11)	LEAQ	"".main·dwrap·1(SB), CX
	0x0056 00086 (main.go:11)	MOVQ	CX, (AX)
	0x0059 00089 (main.go:11)	MOVQ	""..autotmp_4+48(SP), CX
	0x005e 00094 (main.go:11)	TESTB	AL, (CX)
	0x0060 00096 (main.go:11)	MOVQ	""..autotmp_2+40(SP), DX
	0x0065 00101 (main.go:11)	MOVQ	DX, 8(CX)
	0x0069 00105 (main.go:11)	MOVQ	""..autotmp_4+48(SP), CX
	0x006e 00110 (main.go:11)	TESTB	AL, (CX)
	0x0070 00112 (main.go:11)	MOVQ	""..autotmp_3+32(SP), DX
	0x0075 00117 (main.go:11)	MOVQ	DX, 16(CX)
	0x0079 00121 (main.go:11)	MOVQ	""..autotmp_4+48(SP), BX
	0x007e 00126 (main.go:11)	XORL	AX, AX
	0x0080 00128 (main.go:11)	CALL	runtime.newproc(SB) // here
	0x0085 00133 (main.go:12)	MOVQ	56(SP), BP
	0x008a 00138 (main.go:12)	ADDQ	$64, SP
	0x008e 00142 (main.go:12)	RET
	0x008f 00143 (main.go:12)	NOP
	0x008f 00143 (main.go:8)	CALL	runtime.morestack_noctxt(SB)
	0x0094 00148 (main.go:8)	JMP	0
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
```

