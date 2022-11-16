异常与异常捕获
----------------------

有时候，我们不希望程序异常退出，而是希望捕获异常并让函数正常执行，这涉及 defer 和 recover 的结合使用。



### panic 函数使用方法

Go 语言中的 panic 方法并不会导致程序异常退出，而是会终止当前函数的正常执行，并执行 defer 函数并逐级返回。除了手动触发 panic，在 Go 语言运行时的一些阶段也会检查并触发 panic，例如数组越界及 map 并发冲突。



### 异常捕获与 recover

为了让程序在 panic 时仍然能够执行后续的流程，Go 语言提供了内置的 recover 函数用于异常恢复。recover 函数一般与 defer 函数结合使用才有意义，其返回值是 panic 中传递的参数。



### panic 与 recover 的嵌套

重复嵌套的 panic 并不会陷入死循环，每一次的 panic 调用实际都会新建一个 _panic 结构体，并用一个链表进行存储。

```go
func Apple() {
    defer Banana()

    panic("apple")
}

func Banana() {
    defer Cherry()

    panic("banana")
}

func Cherry() {
    panic("cherry")
}

func main() {
    Apple()
    // panic: apple
    // panic: banana
    // panic: cherry
}
```

同时，recover 函数最终捕获的是最近发生的 panic（链表头部），即使有多个 panic 函数，在最上层的函数也只需要一个 recover 函数就能让函数按照正常的流程执行。

```go
func Apple() {
    defer Banana()

    panic("apple")
}

func Banana() {
    defer Cherry()

    panic("banana")
}

func Cherry() {
    panic("cherry")
}

func main() {
    defer func() {
        if err := recover(); err != nil {
            fmt.Println("recover", err)
        }
    }()
    Apple()
    // panic: apple
    // panic: banana
    // panic: cherry
}
```



### panic 函数底层原理

panic 函数在编译时会被解析为调用运行时 runtime.gopanic 函数：

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

每次调用 panic 都会创建一个 _panic 结构体，同时会被放置到 当前协程的链表中，原因是 panic 可能发生嵌套，例如 panic -> defer -> panic，因此可能同时存在多个 _panic 结构体。



### recover 底层原理

在正常情况下，panic 都会遍历 defer 链并退出。当在 defer 中使用了 recover 异常捕获之后。内置的 recover 函数将会在运行时被转换为调用 runtime.goecover 函数：

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

gorecover 函数相对简单，在正常情况下将当前对应的 _panic 结构体的 recovered 字段置为 true 并返回。gorecover 函数的参数 argp 为调用者函数的参数地址，而 p.argp 为发生 panic 时的 defer 函数的参数地址。语句 `argp == uintptr(p.argp)` 的作用是判断 panic 和 recover 是否匹配，防止内层 recover 捕获外层的 panic。

gorecover 并没有进行任何异常处理，真正的处理发生在 runtime.gopanic 函数中，在遍历 defer 链表执行的过程中，一旦发现 p.recovered 被设置为 true，就代表当前 defer 中调用了 recover 函数，会删除当前链表中内联 defer 的 _defer 结构。
