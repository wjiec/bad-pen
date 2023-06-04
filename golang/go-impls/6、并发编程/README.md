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

