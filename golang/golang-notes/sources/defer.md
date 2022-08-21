延迟
-------

延迟调用（defer）的最大优势是，即使函数执行出错，依然能保证回收资源等操作得以执行。但如果对性能有要求，且错误能被控制，那么还是直接执行比较好。

```go
package main

func main() {
    defer func() { println("defer") }()
    println("main")
}
```

通过反汇编可以得到以下内部实现：

```asm
"".main STEXT size=319 args=0x0 locals=0xc8 funcid=0x0
	0x0000 00000 (main.go:14)	TEXT	"".main(SB), ABIInternal, $200-0
	0x0000 00000 (main.go:14)	MOVQ	(TLS), CX
	0x0009 00009 (main.go:14)	LEAQ	-72(SP), AX
	0x000e 00014 (main.go:14)	CMPQ	AX, 16(CX)
	0x0012 00018 (main.go:14)	JLS	309
	0x0018 00024 (main.go:14)	SUBQ	$200, SP
	0x001f 00031 (main.go:14)	MOVQ	BP, 192(SP)
	0x0027 00039 (main.go:14)	LEAQ	192(SP), BP
	0x002f 00047 (main.go:15)	MOVQ	$1, "".v+16(SP)
	0x0038 00056 (main.go:16)	MOVQ	$1, (SP)
	0x0040 00064 (main.go:16)	CALL	runtime.convT64(SB)
	0x0045 00069 (main.go:16)	MOVQ	8(SP), AX
	0x004a 00074 (main.go:16)	MOVQ	AX, ""..autotmp_1+184(SP)
	0x0052 00082 (main.go:16)	LEAQ	type.int(SB), CX
	0x0059 00089 (main.go:16)	MOVQ	CX, (SP)
	0x005d 00093 (main.go:16)	MOVQ	AX, 8(SP)
	0x0062 00098 (main.go:16)	CALL	"".escapes(SB)
	0x0067 00103 (main.go:18)	MOVL	$0, ""..autotmp_2+104(SP)
	0x006f 00111 (main.go:18)	LEAQ	"".main.func1·f(SB), AX
	0x0076 00118 (main.go:18)	MOVQ	AX, ""..autotmp_2+128(SP)
	0x007e 00126 (main.go:18)	LEAQ	""..autotmp_2+104(SP), AX
	0x0083 00131 (main.go:18)	MOVQ	AX, (SP)
	0x0087 00135 (main.go:18)	CALL	runtime.deferprocStack(SB)
	0x008c 00140 (main.go:18)	TESTL	AX, AX
	0x008e 00142 (main.go:18)	JNE	282
	0x0094 00148 (main.go:18)	JMP	150
	0x0096 00150 (main.go:22)	MOVL	$8, ""..autotmp_3+24(SP)
	0x009e 00158 (main.go:22)	LEAQ	"".main.func2·f(SB), AX
	0x00a5 00165 (main.go:22)	MOVQ	AX, ""..autotmp_3+48(SP)
	0x00aa 00170 (main.go:22)	MOVQ	"".v+16(SP), AX
	0x00af 00175 (main.go:22)	MOVQ	AX, ""..autotmp_3+96(SP)
	0x00b4 00180 (main.go:22)	LEAQ	""..autotmp_3+24(SP), AX
	0x00b9 00185 (main.go:22)	MOVQ	AX, (SP)
	0x00bd 00189 (main.go:22)	NOP
	0x00c0 00192 (main.go:22)	CALL	runtime.deferprocStack(SB)
	0x00c5 00197 (main.go:22)	TESTL	AX, AX
	0x00c7 00199 (main.go:22)	JNE	260
	0x00c9 00201 (main.go:22)	JMP	203
	0x00cb 00203 (main.go:26)	CALL	runtime.printlock(SB)
	0x00d0 00208 (main.go:26)	LEAQ	go.string."main\n"(SB), AX
	0x00d7 00215 (main.go:26)	MOVQ	AX, (SP)
	0x00db 00219 (main.go:26)	MOVQ	$5, 8(SP)
	0x00e4 00228 (main.go:26)	CALL	runtime.printstring(SB)
	0x00e9 00233 (main.go:26)	CALL	runtime.printunlock(SB)
	0x00ee 00238 (main.go:27)	XCHGL	AX, AX
	0x00ef 00239 (main.go:27)	CALL	runtime.deferreturn(SB)
	0x00f4 00244 (main.go:27)	MOVQ	192(SP), BP
	0x00fc 00252 (main.go:27)	ADDQ	$200, SP
	0x0103 00259 (main.go:27)	RET
	0x0104 00260 (main.go:22)	XCHGL	AX, AX
	0x0105 00261 (main.go:22)	CALL	runtime.deferreturn(SB)
	0x010a 00266 (main.go:22)	MOVQ	192(SP), BP
	0x0112 00274 (main.go:22)	ADDQ	$200, SP
	0x0119 00281 (main.go:22)	RET
	0x011a 00282 (main.go:18)	XCHGL	AX, AX
	0x011b 00283 (main.go:18)	NOP
	0x0120 00288 (main.go:18)	CALL	runtime.deferreturn(SB)
	0x0125 00293 (main.go:18)	MOVQ	192(SP), BP
	0x012d 00301 (main.go:18)	ADDQ	$200, SP
	0x0134 00308 (main.go:18)	RET
	0x0135 00309 (main.go:18)	NOP
	0x0135 00309 (main.go:14)	PCDATA	$1, $-1
	0x0135 00309 (main.go:14)	PCDATA	$0, $-2
	0x0135 00309 (main.go:14)	CALL	runtime.morestack_noctxt(SB)
	0x013a 00314 (main.go:14)	PCDATA	$0, $-1
	0x013a 00314 (main.go:14)	JMP	0
```

可见，编译器将 defer 处理成两个函数调用：`deferprocStack` 定义一个延迟调用对象（在栈上？），然后在函数结束前通过 `deferreturn` 完成最终的函数调用。

```go
//
// runtime/panic.go
//

// Create a new deferred function fn with siz bytes of arguments.
// The compiler turns a defer statement into a call to this.
//go:nosplit
func deferproc(siz int32, fn *funcval) { // arguments of fn follow fn
	gp := getg()
	if gp.m.curg != gp {
		// go code on the system stack can't defer
		throw("defer on system stack")
	}

	if goexperiment.RegabiDefer && siz != 0 {
		// TODO: Make deferproc just take a func().
		throw("defer with non-empty frame")
	}

	// the arguments of fn are in a perilous state. The stack map
	// for deferproc does not describe them. So we can't let garbage
	// collection or stack copying trigger until we've copied them out
	// to somewhere safe. The memmove below does that.
	// Until the copy completes, we can only call nosplit routines.
	sp := getcallersp()
	argp := uintptr(unsafe.Pointer(&fn)) + unsafe.Sizeof(fn)
	callerpc := getcallerpc()

	d := newdefer(siz)
	if d._panic != nil {
		throw("deferproc: d.panic != nil after newdefer")
	}
	d.link = gp._defer
	gp._defer = d
	d.fn = fn
	d.pc = callerpc
	d.sp = sp
	switch siz {
	case 0:
		// Do nothing.
	case sys.PtrSize:
		*(*uintptr)(deferArgs(d)) = *(*uintptr)(unsafe.Pointer(argp))
	default:
		memmove(deferArgs(d), unsafe.Pointer(argp), uintptr(siz))
	}

	// deferproc returns 0 normally.
	// a deferred func that stops a panic
	// makes the deferproc return 1.
	// the code the compiler generates always
	// checks the return value and jumps to the
	// end of the function if deferproc returns != 0.
	return0()
	// No code can go here - the C return register has
	// been set and must not be clobbered.
}

// deferprocStack queues a new deferred function with a defer record on the stack.
// The defer record must have its siz and fn fields initialized.
// All other fields can contain junk.
// The defer record must be immediately followed in memory by
// the arguments of the defer.
// Nosplit because the arguments on the stack won't be scanned
// until the defer record is spliced into the gp._defer list.
//go:nosplit
func deferprocStack(d *_defer) {
	gp := getg()
	if gp.m.curg != gp {
		// go code on the system stack can't defer
		throw("defer on system stack")
	}
	if goexperiment.RegabiDefer && d.siz != 0 {
		throw("defer with non-empty frame")
	}
	// siz and fn are already set.
	// The other fields are junk on entry to deferprocStack and
	// are initialized here.
	d.started = false
	d.heap = false
	d.openDefer = false
	d.sp = getcallersp()
	d.pc = getcallerpc()
	d.framepc = 0
	d.varp = 0
	// The lines below implement:
	//   d.panic = nil
	//   d.fd = nil
	//   d.link = gp._defer
	//   gp._defer = d
	// But without write barriers. The first three are writes to
	// the stack so they don't need a write barrier, and furthermore
	// are to uninitialized memory, so they must not use a write barrier.
	// The fourth write does not require a write barrier because we
	// explicitly mark all the defer structures, so we don't need to
	// keep track of pointers to them with a write barrier.
	*(*uintptr)(unsafe.Pointer(&d._panic)) = 0
	*(*uintptr)(unsafe.Pointer(&d.fd)) = 0
	*(*uintptr)(unsafe.Pointer(&d.link)) = uintptr(unsafe.Pointer(gp._defer))
	*(*uintptr)(unsafe.Pointer(&gp._defer)) = uintptr(unsafe.Pointer(d))

	return0()
	// No code can go here - the C return register has
	// been set and must not be clobbered.
}

// Run a deferred function if there is one.
// The compiler inserts a call to this at the end of any
// function which calls defer.
// If there is a deferred function, this will call runtime·jmpdefer,
// which will jump to the deferred function such that it appears
// to have been called by the caller of deferreturn at the point
// just before deferreturn was called. The effect is that deferreturn
// is called again and again until there are no more deferred functions.
//
// Declared as nosplit, because the function should not be preempted once we start
// modifying the caller's frame in order to reuse the frame to call the deferred
// function.
//
//go:nosplit
func deferreturn() {
	gp := getg()
	d := gp._defer
	if d == nil {
		return
	}
	sp := getcallersp()
	if d.sp != sp {
		return
	}
	if d.openDefer {
		done := runOpenDeferFrame(gp, d)
		if !done {
			throw("unfinished open-coded defers in deferreturn")
		}
		gp._defer = d.link
		freedefer(d)
		return
	}

	// Moving arguments around.
	//
	// Everything called after this point must be recursively
	// nosplit because the garbage collector won't know the form
	// of the arguments until the jmpdefer can flip the PC over to
	// fn.
	argp := getcallersp() + sys.MinFrameSize
	switch d.siz {
	case 0:
		// Do nothing.
	case sys.PtrSize:
		*(*uintptr)(unsafe.Pointer(argp)) = *(*uintptr)(deferArgs(d))
	default:
		memmove(unsafe.Pointer(argp), deferArgs(d), uintptr(d.siz))
	}
	fn := d.fn
	d.fn = nil
	gp._defer = d.link
	freedefer(d)
	// If the defer function pointer is nil, force the seg fault to happen
	// here rather than in jmpdefer. gentraceback() throws an error if it is
	// called with a callback on an LR architecture and jmpdefer is on the
	// stack, because the stack trace can be incorrect in that case - see
	// issue #8153).
	_ = fn.fn
	jmpdefer(fn, argp)
}
```



### panic

编译器将 panic 翻译成 gopanic 函数调用。它会将错误信息打包成 _panic 对象，并挂到 G.\_panic 链表的头部，然后遍历执行 G.\_defer 链表，检查是否 recover。如被 recover 则终止遍历执行，跳转到正常的 deferreturn 环节。否则执行整个调用堆栈的延迟函数后，显示异常信息，最后终止进程。

```go
// The implementation of the predeclared function panic.
func gopanic(e interface{}) {
	gp := getg()
	if gp.m.curg != gp {
		print("panic: ")
		printany(e)
		print("\n")
		throw("panic on system stack")
	}

	if gp.m.mallocing != 0 {
		print("panic: ")
		printany(e)
		print("\n")
		throw("panic during malloc")
	}
	if gp.m.preemptoff != "" {
		print("panic: ")
		printany(e)
		print("\n")
		print("preempt off reason: ")
		print(gp.m.preemptoff)
		print("\n")
		throw("panic during preemptoff")
	}
	if gp.m.locks != 0 {
		print("panic: ")
		printany(e)
		print("\n")
		throw("panic holding locks")
	}

	var p _panic
	p.arg = e
	p.link = gp._panic
	gp._panic = (*_panic)(noescape(unsafe.Pointer(&p)))

	atomic.Xadd(&runningPanicDefers, 1)

	// By calculating getcallerpc/getcallersp here, we avoid scanning the
	// gopanic frame (stack scanning is slow...)
	addOneOpenDeferFrame(gp, getcallerpc(), unsafe.Pointer(getcallersp()))

	for {
		d := gp._defer
		if d == nil {
			break
		}

		// If defer was started by earlier panic or Goexit (and, since we're back here, that triggered a new panic),
		// take defer off list. An earlier panic will not continue running, but we will make sure below that an
		// earlier Goexit does continue running.
		if d.started {
			if d._panic != nil {
				d._panic.aborted = true
			}
			d._panic = nil
			if !d.openDefer {
				// For open-coded defers, we need to process the
				// defer again, in case there are any other defers
				// to call in the frame (not including the defer
				// call that caused the panic).
				d.fn = nil
				gp._defer = d.link
				freedefer(d)
				continue
			}
		}

		// Mark defer as started, but keep on list, so that traceback
		// can find and update the defer's argument frame if stack growth
		// or a garbage collection happens before executing d.fn.
		d.started = true

		// Record the panic that is running the defer.
		// If there is a new panic during the deferred call, that panic
		// will find d in the list and will mark d._panic (this panic) aborted.
		d._panic = (*_panic)(noescape(unsafe.Pointer(&p)))

		done := true
		if d.openDefer {
			done = runOpenDeferFrame(gp, d)
			if done && !d._panic.recovered {
				addOneOpenDeferFrame(gp, 0, nil)
			}
		} else {
			p.argp = unsafe.Pointer(getargp())

			if goexperiment.RegabiDefer {
				fn := deferFunc(d)
				fn()
			} else {
				// Pass a dummy RegArgs since we'll only take this path if
				// we're not using the register ABI.
				var regs abi.RegArgs
				reflectcall(nil, unsafe.Pointer(d.fn), deferArgs(d), uint32(d.siz), uint32(d.siz), uint32(d.siz), &regs)
			}
		}
		p.argp = nil

		// Deferred function did not panic. Remove d.
		if gp._defer != d {
			throw("bad defer entry in panic")
		}
		d._panic = nil

		// trigger shrinkage to test stack copy. See stack_test.go:TestStackPanic
		//GC()

		pc := d.pc
		sp := unsafe.Pointer(d.sp) // must be pointer so it gets adjusted during stack copy
		if done {
			d.fn = nil
			gp._defer = d.link
			freedefer(d)
		}
		if p.recovered {
			gp._panic = p.link
			if gp._panic != nil && gp._panic.goexit && gp._panic.aborted {
				// A normal recover would bypass/abort the Goexit.  Instead,
				// we return to the processing loop of the Goexit.
				gp.sigcode0 = uintptr(gp._panic.sp)
				gp.sigcode1 = uintptr(gp._panic.pc)
				mcall(recovery)
				throw("bypassed recovery failed") // mcall should not return
			}
			atomic.Xadd(&runningPanicDefers, -1)

			// Remove any remaining non-started, open-coded
			// defer entries after a recover, since the
			// corresponding defers will be executed normally
			// (inline). Any such entry will become stale once
			// we run the corresponding defers inline and exit
			// the associated stack frame.
			d := gp._defer
			var prev *_defer
			if !done {
				// Skip our current frame, if not done. It is
				// needed to complete any remaining defers in
				// deferreturn()
				prev = d
				d = d.link
			}
			for d != nil {
				if d.started {
					// This defer is started but we
					// are in the middle of a
					// defer-panic-recover inside of
					// it, so don't remove it or any
					// further defer entries
					break
				}
				if d.openDefer {
					if prev == nil {
						gp._defer = d.link
					} else {
						prev.link = d.link
					}
					newd := d.link
					freedefer(d)
					d = newd
				} else {
					prev = d
					d = d.link
				}
			}

			gp._panic = p.link
			// Aborted panics are marked but remain on the g.panic list.
			// Remove them from the list.
			for gp._panic != nil && gp._panic.aborted {
				gp._panic = gp._panic.link
			}
			if gp._panic == nil { // must be done with signal
				gp.sig = 0
			}
			// Pass information about recovering frame to recovery.
			gp.sigcode0 = uintptr(sp)
			gp.sigcode1 = pc
			mcall(recovery)
			throw("recovery failed") // mcall should not return
		}
	}

	// ran out of deferred calls - old-school panic now
	// Because it is unsafe to call arbitrary user code after freezing
	// the world, we call preprintpanics to invoke all necessary Error
	// and String methods to prepare the panic strings before startpanic.
	preprintpanics(gp._panic)

	fatalpanic(gp._panic) // should not return
	*(*int)(nil) = 0      // not reached
}
```

和 panic 相比， recover 函数除返回最后一个错误信息外，主要是设置 recovered 标志。注意，它会通过参数堆栈地址确认时候在延迟调用函数内被直接调用。

```go
// The implementation of the predeclared function recover.
// Cannot split the stack because it needs to reliably
// find the stack segment of its caller.
//
// TODO(rsc): Once we commit to CopyStackAlways,
// this doesn't need to be nosplit.
//go:nosplit
func gorecover(argp uintptr) interface{} {
	// Must be in a function running as part of a deferred call during the panic.
	// Must be called from the topmost function of the call
	// (the function used in the defer statement).
	// p.argp is the argument pointer of that topmost deferred function call.
	// Compare against argp reported by caller.
	// If they match, the caller is the one who can recover.
	gp := getg()
	p := gp._panic
	if p != nil && !p.goexit && !p.recovered && argp == uintptr(p.argp) {
		p.recovered = true
		return p.arg
	}
	return nil
}
```

