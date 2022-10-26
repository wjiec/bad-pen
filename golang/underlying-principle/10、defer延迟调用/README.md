defer 延迟调用
---------------------

 defer 是 Go 语言中的关键字，也是 Go 语言的重要特性之一。defer 之后必须紧跟一个函数调用或者方法调用，且不能用括号括起来。在很多时候，defer 后的函数都是以匿名函数或闭包的形式呈现。

defer 将其后的函数推迟到其所在函数返回之前执行，不管 defer 语句后的执行路径如何，最终都将被执行。在 Go 语言中，defer 一般被用于资源的释放及异常 panic 的捕获处理。



### 使用 defer 的优势



#### 释放资源

作为 Go 语言的特性之一，defer 给 Go 代码的编写方式带来了很大的变化：

```go
func CopyFile(dst, src string) (int64, error) {
    sfp, err := os.Open(src)
    if err != nil {
        return 0, err
    }
    defer func() { _ = src.Close() }()
    
    dfp, err := os.Create(dst)
    if err != nil {
        return 0, err
    }
    defer func() { _ = dfp.Close() }()

    return io.Copy(dfp, sfp)
}
```

defer 是一种优雅的关闭资源的方式，能减少大量冗余的代码并避免由于忘记释放资源而产生的错误。



#### 异常捕获

程序在运行时可能在任意的地方发生 panic 异常，同时这些错误会导致程序异常退出。在很多时候，我们希望能够捕获这样的错误，同时希望程序能够继续正常执行。而 defer 为异常补货提供了很好的时机，一般和 recover 函数结合在一起使用。



### defer 特性

defer 后的函数不会立即执行，而是推迟到函数结束后再执行。这一特性一般用于资源的释放，除了可以用在资源释放和异常捕获上，有时也可以用于函数的中间件。



#### 参数预计算

defer 在使用时，延迟调用函数的参数将立即求值，传递到 defer 函数中的参数将预先被固定，而不会等到函数执行完成后再传递到 defer 函数中。

```go
func main() {
    i := 100
    
    defer func(n int) {
        fmt.Printf("defer n = %d\n", n)
    }(i + 1)
    
    i = 200
    fmt.Printf("main i = %d\n", i)
    // main i = 200
    // defer n = 101
}
```



#### defer 多次执行与 LIFO 执行顺序

在函数体内部出现的多个 defer 函数将会按照后入先出（Last-In First-Out）的顺序执行。

```go
func main() {
    defer func() { fmt.Println("hello") }()
    defer func() { fmt.Println("world") }()
    defer func() { fmt.Println("foo") }()
    defer func() { fmt.Println("bar") }()
    // bar
    // foo
    // world
    // hello
}
```



#### 返回值陷阱

当 defer 与返回值相结合时，需要注意返回语句的语义问题：

```go
var g = 100

func f1() (r int) {
    defer func() { g = 200 }()

    return g
}

func f2() (r int) {
    defer func() { r = 300 }()

    return g
}

func main() {
    fmt.Printf("f1() = %d\n", f1())
    fmt.Printf("f2() = %d\n", f2())
    // f1() = 100
    // f2() = 300
}
```

以上问题的原因在于，return 其实并不是一个原子操作，其包含了以下几个步骤：

* 将返回值保存到栈上
* 执行 defer 语句
* 函数执行 RET 返回



### defer 底层原理

在 Go 1.13 之前，defer 是被分配到堆区的，尽管有全局的缓存池分配，仍然有比较大的性能问题，原因在于使用 defer 不仅涉及堆内存的分配，在一开始还需要存储 defer 函数中的参数，最后还需要将堆区的数据转移到栈中执行，这涉及到内存的复制。

为了降低 defer 函数的调用开销，Go 1.13 在大部分情况下将 defer 语句放置在栈中，避免在堆区分配、复制对象。但是仍然需要将 defer 对象放置到一个链表中，以保证 LIFO 的顺序执行。

在 Go 1.13 中包括两种策略，对于最多调用一个（at most once）语义的 defer 语句使用了栈分配的策略，而对于其他的方式，例如 for 循环体内部的 defer 语句，仍然采用之前的堆分配策略。Go 1.14 则更进一步对最多调用一次的 defer 语义进行了优化，通过编译时实现内联优化。



#### 堆分配

在大部分情况下，堆分配的 defer 对象只会在循环结构中出现，例如：

```go
func stack() {
    for i := 0; i < 3; i++ {
        defer func() { fmt.Println("defer func") }()
    }
}
```

经过反汇编之后，我们可以发现每一条 defer 语句都对应一个 `runtime.deferproc` 调用，而在函数退出前，则会调用 `runtime.deferreturn` 函数。其中 `runtime.deferproc` 的流程比较简单：

* 计算 `deferproc` 调用者的 SP、PC 寄存器值以及存放在栈中的位置
* 在堆内存中分配 _defer 结构体，并将其插入当前协程记录 _defer 的链表头部
* 将 SP、PC 寄存器值记录到新的 _defer 结构体中，并将栈上的参数复制到堆区

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
```



每个协程在运行时的表示为一个 g 结构体，deferproc 函数创建的 _defer 结构最终会被放置到当前协程存储 _defer 结构的链表中：

```go
//
// runtime/runtime2.go
//

type g struct {
    // ...
	_defer    *_defer // innermost defer
    // ...
}
```

新加入的 _defer 结构会被放置到当前链表的头部，从而保证在后续执行 defer 函数时能以先入后出的顺序执行。



##### runtime.newdefer

runtime.newdefer 在堆中申请具体的 _defer 结构体，每个逻辑处理器 P 中都有局部缓存（deferpool），在全局中也有一个缓存池（schedt.deferpool）。当全局缓存和局部缓存池中都搜索不到对象时，需要在堆区分配指定大小的 _defer。

```go
// Allocate a Defer, usually using per-P pool.
// Each defer must be released with freedefer.  The defer is not
// added to any defer chain yet.
//
// This must not grow the stack because there may be a frame without
// stack map information when this is called.
//
//go:nosplit
func newdefer(siz int32) *_defer {
	var d *_defer
	sc := deferclass(uintptr(siz))
	gp := getg()
	if sc < uintptr(len(p{}.deferpool)) {
		pp := gp.m.p.ptr()
		if len(pp.deferpool[sc]) == 0 && sched.deferpool[sc] != nil {
			// Take the slow path on the system stack so
			// we don't grow newdefer's stack.
			systemstack(func() {
				lock(&sched.deferlock)
				for len(pp.deferpool[sc]) < cap(pp.deferpool[sc])/2 && sched.deferpool[sc] != nil {
					d := sched.deferpool[sc]
					sched.deferpool[sc] = d.link
					d.link = nil
					pp.deferpool[sc] = append(pp.deferpool[sc], d)
				}
				unlock(&sched.deferlock)
			})
		}
		if n := len(pp.deferpool[sc]); n > 0 {
			d = pp.deferpool[sc][n-1]
			pp.deferpool[sc][n-1] = nil
			pp.deferpool[sc] = pp.deferpool[sc][:n-1]
		}
	}
	if d == nil {
		// Allocate new defer+args.
		systemstack(func() {
			total := roundupsize(totaldefersize(uintptr(siz)))
			d = (*_defer)(mallocgc(total, deferType, true))
		})
	}
	d.siz = siz
	d.heap = true
	return d
}
```

当 defer 执行完毕被销毁后，会重新回到局部缓存池中，当局部缓存池容纳了足够的对象时，会将 _defer 结构体放入全局缓存池。存储在全局和局部缓存池中的对象如果没有被使用，则最终在垃圾回收阶段被销毁。



#### deferreturn 调用

在函数正常结束时，其递归调用了 runtime.deferreturn 函数，在该函数中会遍历 defer 链表，并调用存储在 defer 中的函数。

```go
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

deferreturn 函数会检查当前函数的 sp 寄存器地址，如果 sp 寄存器不同，则说明不是当前函数的 defer 调用了，需要退出。当 deferreturn 获取需要执行的函数之后，需要将当前 defer 函数的参数重新转移到栈中，调用 freedefer 销毁当前的结构体，并将链表指向下一个 defer 结构体。最终通过 jmpdefer 函数完成调用：

```go
// func jmpdefer(fv *funcval, argp uintptr)
// argp is a caller SP.
// called from deferreturn.
// 1. pop the caller
// 2. sub 5 bytes from the callers return
// 3. jmp to the argument
TEXT runtime·jmpdefer(SB), NOSPLIT, $0-16
	MOVQ	fv+0(FP), DX	// fn
	MOVQ	argp+8(FP), BX	// caller sp
	LEAQ	-8(BX), SP	// caller sp after CALL
	MOVQ	-8(SP), BP	// restore BP as if deferreturn returned (harmless if framepointers not in use)
	SUBQ	$5, (SP)	// return to CALL again
	MOVQ	0(DX), BX
	JMP	BX	// but first run the deferred function
```

jmpdefer 函数使用了比较巧妙的方式实现了对 deferreturn 函数的反复调用。其核心思想是调整了 deferreturn 函数的 SP、BP 地址，使 deferreturn 函数退出之后再次调用 deferreturn 函数，从而实现循环调用。

**由于 jmpdefer 函数在执行完毕返回时可以递归调用 deferreturn 函数，复用了栈空间，所以不会因为大量调用导致栈溢出。**



#### Go 1.13 栈分配优化

Go 1.13 为了解决堆分配的效率问题，对于最多调用一次的 defer 语义采用了在栈中分配的策略。当执行到 defer 语句时，调用都会变为执行运行时的 runtime.deferprocStack 函数。在函数的最后，和堆分配一样，仍然插入了 runtime.deferreturn 函数用于遍历调用链：

```go
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
```

该函数传递了一个 _defer 指针，该 _defer 其实已经放置在栈中了。并在执行前将 defer 的大小、参数、哈数指针放置在了栈中，在 deferprocStack 中只需要获取必要的调用者 SP、PC 指针并将 _defer 压入链表的头部。



#### Go 1.14 内联优化

虽然 Go 1.13 中 defer 的栈策略已经有比较大的优化，但是与直接的函数调用相比还是有很大差距。一种容易想到的优化策略是在编译时在函数结束前直接调用 defer 函数。这样还可以省去放置到 _defer 链表和遍历的时间。

采用这种方式最大的困难在于 defer 函数并不一定能够执行（条件选择的 defer 函数）。为了解决这个的问题，Go 语言编译器采取了一种巧妙的方式。通过在栈中初始化 1 字节的临时变量，以位图的形式来判断函数是否需要执行。
