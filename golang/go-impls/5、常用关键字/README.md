常用关键字
---------------

关键字具有非常特殊的含义，它们是编程语言对外提供的接口的一部分。



### for 和 range

除了经典的 for 三段式循环外，Go 语言还引入了另一个关键字 range 帮助我们快速遍历数组、切片、哈希表和 channel 等集合类型。

#### 经典循环

经典循环在编译器看起来是一个 `OFOR` 类型的节点，该节点由以下 4 部分组成：

* 初始化循环的 `Ninit`
* 循环的继续条件 `Left`
* 循环体结束时执行的 `Right`
* 循环体 `NBody`

#### 范围循环

在编译器期间，编译器会将所有 for-range 循环变为经典循环。节点的转换过程发生在中间代码生成阶段，所有的 for-range 循环都会被编译器转换成不包含复杂结构、只包含基本表达式的语句。

##### 数组与切片

**对于所有的 range 循环，Go 语言都会在编译器将原切片或者数组赋值给一个新变量（发生了复制）**

##### 哈希表

在遍历哈希表时，编译器会使用 `runtime.mapiterinit` 和 `runtime.mapiternext` 两个运行时函数重写原始的 for-range 循环。

##### 字符串

遍历字符串的过程与遍历数组、切片和哈希表非常相似，只是在遍历时会获取字符串中的索引和对应字节并将其转换成 rune 类型。

##### channel

该循环会使用 `<-ch` 从 channel 中取出待处理的值，这个操作会调用 `runtime.chanrecv2` 并阻塞当前协程，当 `runtime.chanrecv2` 返回时会根据布尔值判断是否需要跳出循环。



### select

Go 语言中的 `select` 能够让 Goroutine 同时等待多个 channel 可读或者可写。在 Go 语言中使用 `select` 控制结构时，我们有：

* `select` 能在 channel 上进行非阻塞的收发操作（default 子句）
* `select` 在遇到多个 channel 同时响应时，会随机选择执行一个分支

#### 实现原理

`select` 语句在编译期间会被转换成 `OSELECT` 节点，在生成中间代码期间，会根据 `select` 中 `case` 的不同对控制语句进行优化：

* 不存在任何 case
* 只存在一个 case
* 存在两个 case，其中一个是 default
* 存在多个 case

##### 直接阻塞

当 `select` 结构中不包含任何 case，编译器会将 `select {}` 语句直接转换成调用 `runtime.block` 函数。此时 Goroutine 进入无法被唤醒的永久休眠状态。

##### 单一 channel

如果当前 `select` 控制结构中只包含一个 case，那么编译器会将 select 改写成 if 条件语句。

##### 非阻塞操作

当 `select` 中包含两个 case 且其中一个是 default 的情况下，编译器会认为这是一次非阻塞收发操作。

发送情况：会使用条件语句和 `runtime.selectnbsend` 函数改写代码

接收情况：根据返回值数量的不同，会被改写成 `runtime.selectnbrecv` 和 `runtime.selctnbrecv2` 函数。

##### 常用流程

编译器使用如下流程处理多分支的 `select` 语句

* 将所有 case 转换成包含 channel 以及类型等信息的 `runtime.scase` 结构体
* 调用运行时 `runtime.selectgo` 从多个准备就绪的 channel 中选择一个可执行的 `runtime.scase` 结构体
* 通过 for 循环生成一组 if 语句，在语句中判断自己是不是被选中的 case。

随机顺序可以避免 channel 的饥饿问题，保证公平性。根据 channel 的地址顺序进行加锁能够避免死锁。

在 `runtime.selectgo` 中有三个阶段：

* 查找所有 case 中是否有可以立刻被处理的 channel
* 按需将 channel 加入到 sendq 和 recvq 队列中
* 最后从 `runtime.sudog` 中读取数据



### defer

在 Go 语言中，defer 的实现是由编译器和运行时共同完成的。向 defer 关键字传入的函数会在函数返回之前运行，在**使用 defer 关键字时会立刻复制函数中引用的外部参数**。

#### 数据结构

defer 关键字在 Go 语言运行时对应的数据结构是 `runtime._defer` 结构体：

```go
//
// runtime/runtime2.go
//

// A _defer holds an entry on the list of deferred calls.
// If you add a field here, add code to clear it in deferProcStack.
// This struct must match the code in cmd/compile/internal/ssagen/ssa.go:deferstruct
// and cmd/compile/internal/ssagen/ssa.go:(*state).call.
// Some defers will be allocated on the stack and some on the heap.
// All defers are logically part of the stack, so write barriers to
// initialize them are not required. All defers must be manually scanned,
// and for heap defers, marked.
type _defer struct {
	started bool
	heap    bool
	// openDefer indicates that this _defer is for a frame with open-coded
	// defers. We have only one defer record for the entire frame (which may
	// currently have 0, 1, or more defers active).
	openDefer bool
	sp        uintptr // sp at time of defer
	pc        uintptr // pc at time of defer
	fn        func()  // can be nil for open-coded defers
	_panic    *_panic // panic that is running defer
	link      *_defer // next defer on G; can point to either heap or stack!

	// If openDefer is true, the fields below record values about the stack
	// frame and associated function that has the open-coded defer(s). sp
	// above will be the sp for the frame, and pc will be address of the
	// deferreturn call in the function.
	fd   unsafe.Pointer // funcdata for the function associated with the frame
	varp uintptr        // value of varp for the stack frame
	// framepc is the current pc associated with the stack frame. Together,
	// with sp above (which is the sp associated with the stack frame),
	// framepc/sp can be used as pc/sp pair to continue a stack trace via
	// gentraceback().
	framepc uintptr
}
```

`runtime._defer` 结构体是延迟调用链表上的一个元素，所有结构体都会通过 `link` 字段串联成链表。defer 关键字的插入顺序是从后往前，而 defer 关键字执行是从前往后的，这也是为什么后调用的 defer 会优先执行。

#### 堆中分配

堆中分配是 `runtime._defer` 结构体是默认的兜底方案，编译器会将 defer 关键字都转换为 `runtime.deferproc` 函数，还会在函数返回之前插入 `runtime.deferreturn` 的函数调用。

#### 栈上分配

Go 1.13 对 defer 关键字进行了优化，当该关键字在函数体中最多执行一次时，编译器会将结构体分配到栈上并调用 `runtime.deferprocStack` 函数。与堆中分配的 `runtime._defer` 相比，该方法可以将 defer 关键字的额外开销降低越 30%。

#### 开放编码

Go 1.14 通过开放编码实现 defer 关键字，该设计使用代码内联优化 ddefer 关键字的额外开销，可以将 defer 的调用开销从 35ns 降低到 6ns。开发编码作为一种优化 defer 的方法，只有在以下场景会启用：

* 函数的 defer 少于或等于 8 个（`deferBits`）
* 函数的 defer 关键字不能在循环中执行
* 函数的 return 语句与 defer 语句的乘积小于等于 15

延迟比特和延迟记录是使用开放编码实现 defer 的两个最重要的结构，一旦决定使用开放编码，编译器会在编译期间在栈上初始化大小为 8 比特的 `deferBits` 变量。

延迟比特的作用就是标记哪些 `defer` 关键字在函数中被执行，这样在函数返回时就可以跟酒对应 `deferBits` 的内容确定需要执行的函数。而也是因为 `deferBits` 的大小仅为 8 比特，所以该优化的启用条件为函数中的 defer 关键字数量少于 8 个。



### panic 和 recover

panic 能够改变程序的控制流，调用 panic 后会立刻停止执行当前函数的剩余代码，并在当前 Goroutine 中递归执行调用方的 defer 函数。而 recover 可以中止 panic 造成的程序崩溃。其中有几个重点：

* panic 只会触发当前 Goroutine 的 defer 函数
* recover 只在 defer 中调用中生效
* panic 允许在 defer 中嵌套多次调用

#### 数据结构

在运行时，panic 由 `runtime._panic` 所表示，编译器会将 panic 调用转换为 `runtime.gopanic` 函数：

```go
//
// runtime/runtime2.go
//

// A _panic holds information about an active panic.
//
// A _panic value must only ever live on the stack.
//
// The argp and link fields are stack pointers, but don't need special
// handling during stack growth: because they are pointer-typed and
// _panic values only live on the stack, regular stack pointer
// adjustment takes care of them.
type _panic struct {
	argp      unsafe.Pointer // pointer to arguments of deferred call run during panic; cannot move - known to liblink
	arg       any            // argument to panic
	link      *_panic        // link to earlier panic
	pc        uintptr        // where to return to in runtime if this panic is bypassed
	sp        unsafe.Pointer // where to return to in runtime if this panic is bypassed
	recovered bool           // whether this panic is over
	aborted   bool           // the panic was aborted
	goexit    bool
}
```

#### 崩溃恢复

编译器将关键字 recover 转换为 `runtime.gorecover` 函数调用：

```go
//
// runtime/panic.go
//

// The implementation of the predeclared function recover.
// Cannot split the stack because it needs to reliably
// find the stack segment of its caller.
//
// TODO(rsc): Once we commit to CopyStackAlways,
// this doesn't need to be nosplit.
//
//go:nosplit
func gorecover(argp uintptr) any {
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



### make 和 new

make 的作用是初始化内置的数据结构，new 的作用是根据传入的类型分配一块内存空间，并返回指向这块内存空间的指针。
