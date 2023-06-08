并发编程
------------

并发编程是 Go 语言中最重要也是最迷人的部分，Go 语言为应用程序带来的高并发和高性能都源于此。



### 上下文

上下文 `context.Context` 是用来设置截止日期、同步信号、传递请求相关值的结构体。`context.Context` 的最大作用是，在 Goroutine 构成的树形结构中同步信号，达到减少计算资源的浪费。



### 同步原语和锁

锁是并发编程中的一种同步原语，它能保证多个 Goroutine 在访问同一块内存时不会出现竞争等问题。

#### Mutex

Go 中的 `sync.Mutex` 由 `state` 和 `sema` 两个字短组成，其中 `state` 表示当前互斥锁的状态，而 `sema` 是用于控制锁状态的信号量。

```go
// A Mutex is a mutual exclusion lock.
// The zero value for a Mutex is an unlocked mutex.
//
// A Mutex must not be copied after first use.
//
// In the terminology of the Go memory model,
// the n'th call to Unlock “synchronizes before” the m'th call to Lock
// for any n < m.
// A successful call to TryLock is equivalent to a call to Lock.
// A failed call to TryLock does not establish any “synchronizes before”
// relation at all.
type Mutex struct {
	state int32
	sema  uint32
}
```

互斥锁中的 `state` 状态，最低 3 位分别表示 `mutexLocked`，`mutexWoken`，`mutexStarving`，剩余位置用来表示当前还有多少 Goroutine 在等待互斥锁的释放。

* `mutexLocked`：互斥锁的锁定状态
* `mutexWoken`：是否从正常模式唤醒
* `mutexStarving`：是否进入饥饿模式

互斥锁由两种模式——正常模式和饥饿模式。

* 正常模式：锁的等待者回按照新进先出的顺序获得锁。**能提供更好的性能**
* 饥饿模式：锁会直接交给等待队列最前面的 Goroutine。**避免无法获取锁而陷入等待造成高尾延迟**

##### 加锁与解锁

互斥锁的加锁涉及自旋、信号量以及调度等概念。解锁则比较简单，根据所处的模式执行相应的逻辑即可。

#### RWMutex

`sync.RWMutex` 是细粒度的互斥锁，它不限制资源的并发读，但是写操作无法并发进行。它建立在 `sync.Mutex` 上，在获取读锁时回使用 `sync.Mutex` 执行锁定，而获取读锁时会增加 `sync.Mutex.readerCount` 的值。



### 计时器

Go 语言使用四叉堆维护所有的计时器。在 Go 1.10 之前，所有的计时器共用全局唯一的四叉堆，计时器的各种操作都需要获取全局唯一的互斥锁，这会严重影响计时器的性能。

Go 1.10 将全局的四叉堆分割成了 64 个小的四叉堆，虽然增加了内存的占有同时能降低锁的粒度，但是计时器造成的处理器与线程之间频繁的上下文切换成了影响计时器性能的首要因素。

在最新版本的实现中移除了计时器桶，所有的计时器都以最小四叉堆的形式存储在处理器 `runtime.p` 中。

#### 数据结构

`runtime.timer` 是 Go 语言计时器的内部表示：

```go
//
// runtime/time.go
//

// Package time knows the layout of this structure.
// If this struct changes, adjust ../time/sleep.go:/runtimeTimer.
type timer struct {
	// If this timer is on a heap, which P's heap it is on.
	// puintptr rather than *p to match uintptr in the versions
	// of this struct defined in other packages.
	pp puintptr

	// Timer wakes up at when, and then at when+period, ... (period > 0 only)
	// each time calling f(arg, now) in the timer goroutine, so f must be
	// a well-behaved function and not block.
	//
	// when must be positive on an active timer.
	when   int64 // 当前计时器被唤醒的时间
	period int64 // 两次被唤醒的间隔
	f      func(any, uintptr) // 每次唤醒都会调用的函数
	arg    any // 每次唤醒调用 f 传入的参数
	seq    uintptr // 计时器被唤醒调用 f 时传入的参数，与 netpoll 有关

	// What to set the when field to in timerModifiedXX status.
	nextwhen int64 // 当修改计时器时需要修改到的时间

	// The status field holds one of the values below.
	status atomic.Uint32 // 计时器的状态
}
```

对外暴露的计时器使用 `time.Timer` 结构体：

```go
// The Timer type represents a single event.
// When the Timer expires, the current time will be sent on C,
// unless the Timer was created by AfterFunc.
// A Timer must be created with NewTimer or AfterFunc.
type Timer struct {
	C <-chan Time
	r runtimeTimer
}
```

#### 触发计时器

Go 语言会在以下时刻触发计时器，运行计时器中保存的函数：

* 调度器调度时会检查处理器中的计时器是否准备就绪（`runtime.checkTimers`）
* 系统监控会检查是否有未执行的到期计时器（`runtime.sysmon`）



### Channel

