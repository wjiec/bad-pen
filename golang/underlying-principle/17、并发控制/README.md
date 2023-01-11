并发控制
------------

> Data races are among the most common and hardest to debug types of bugs in concurrent system.



### context

使用 context 最重要的一点是协程之间时常存在着级联关系，退出需要具有传递性。



#### context 的实现原理

context 在很大程度上利用了通道在 close 时会通知所有监听它的协程这一特性来实现。每个派生出的子上下文都会创建一个新的退出通道，组织好 context 之间的关闭即可实现继承链上退出信号的传递。



### 数据竞争检查

数据竞争指在 Go 语言中两个协程同时访问相同的内存空间，并且至少有一个写操作的情况。



#### 数据竞争工具

Go 1.1 之后提供了强大的检查工具 race 来排查数据竞争问题。当检测器在程序中找到数据竞争时，将打印报告。该报告包含发生 race 冲突的协程栈，以及此时正在运行的协程栈。



#### race 工具原理

race 工具借助了 `ThreadSanitizer` 来实现的， `ThreadSanitizer` 是谷歌为了应对内部大量服务端 C++ 代码的数据竞争问题而开发的新一代工具，目前也被 Go 语言通过 CGO 的形式进行调用。

另外 race 工具还借助矢量时钟（Vector Clock）技术用来观察事件之间 happened-before 的顺序，该技术在分布式系统中使用广泛，用于检测和确定分布式系统中事件的因果关系，也可以用于数据竞争的探测。



### 锁

通过锁来保证某一个时刻只有一个协程可以执行特定操作。



#### 原子锁

原子锁的实现通常依赖于硬件的支持，例如 X86 指令集中的 LOCK 指令，对应 Go 语言中的 `async/tomic` 包。原子操作是底层最基础的同步保证，通过原子操作可以构建起许多同步原语。例如自旋锁、信号量、互斥锁等。



#### 互斥锁

直接通过原子操作构建的互斥锁虽然高效且简单，但是其并不是万能的。当同时有许多正在获取锁的协程时，可能有协程一直抢占不到锁。

在 Go 语言中的互斥锁是一种混合锁，其实现方式包含了自旋锁，同时参考了操作系统锁的实现。

为了解决某一个协程可能长时间无法获取锁的问题，Go 1.9 之后使用了饥饿模式。在饥饿模式下，unlock 会直接唤醒最先申请加速的协程，从而保证公平。

##### 互斥锁的获取

互斥锁的第一阶段是使用原子操作快速抢占锁，如果抢占成功则立即返回，如果抢占失败则调用 `lockSlow` 方法：

```go
// Lock locks m.
// If the lock is already in use, the calling goroutine
// blocks until the mutex is available.
func (m *Mutex) Lock() {
	// Fast path: grab unlocked mutex.
	if atomic.CompareAndSwapInt32(&m.state, 0, mutexLocked) {
		if race.Enabled {
			race.Acquire(unsafe.Pointer(m))
		}
		return
	}
	// Slow path (outlined so that the fast path can be inlined)
	m.lockSlow()
}
```

在 `lockSlow` 中会自旋尝试抢占锁一段时间，而不会立即进入休眠状态，这使得互斥锁在频繁加锁与释放锁时也能良好工作。

```go
func (m *Mutex) lockSlow() {
	var waitStartTime int64
	starving := false
	awoke := false
	iter := 0
	old := m.state
	for {
		// Don't spin in starvation mode, ownership is handed off to waiters
		// so we won't be able to acquire the mutex anyway.
		if old&(mutexLocked|mutexStarving) == mutexLocked && runtime_canSpin(iter) {
			// Active spinning makes sense.
			// Try to set mutexWoken flag to inform Unlock
			// to not wake other blocked goroutines.
			if !awoke && old&mutexWoken == 0 && old>>mutexWaiterShift != 0 &&
				atomic.CompareAndSwapInt32(&m.state, old, old|mutexWoken) {
				awoke = true
			}
			runtime_doSpin()
			iter++
			old = m.state
			continue
		}
		new := old
		// Don't try to acquire starving mutex, new arriving goroutines must queue.
		if old&mutexStarving == 0 {
			new |= mutexLocked
		}
		if old&(mutexLocked|mutexStarving) != 0 {
			new += 1 << mutexWaiterShift
		}
		// The current goroutine switches mutex to starvation mode.
		// But if the mutex is currently unlocked, don't do the switch.
		// Unlock expects that starving mutex has waiters, which will not
		// be true in this case.
		if starving && old&mutexLocked != 0 {
			new |= mutexStarving
		}
		if awoke {
			// The goroutine has been woken from sleep,
			// so we need to reset the flag in either case.
			if new&mutexWoken == 0 {
				throw("sync: inconsistent mutex state")
			}
			new &^= mutexWoken
		}
		if atomic.CompareAndSwapInt32(&m.state, old, new) {
			if old&(mutexLocked|mutexStarving) == 0 {
				break // locked the mutex with CAS
			}
			// If we were already waiting before, queue at the front of the queue.
			queueLifo := waitStartTime != 0
			if waitStartTime == 0 {
				waitStartTime = runtime_nanotime()
			}
			runtime_SemacquireMutex(&m.sema, queueLifo, 1)
			starving = starving || runtime_nanotime()-waitStartTime > starvationThresholdNs
			old = m.state
			if old&mutexStarving != 0 {
				// If this goroutine was woken and mutex is in starvation mode,
				// ownership was handed off to us but mutex is in somewhat
				// inconsistent state: mutexLocked is not set and we are still
				// accounted as waiter. Fix that.
				if old&(mutexLocked|mutexWoken) != 0 || old>>mutexWaiterShift == 0 {
					throw("sync: inconsistent mutex state")
				}
				delta := int32(mutexLocked - 1<<mutexWaiterShift)
				if !starving || old>>mutexWaiterShift == 1 {
					// Exit starvation mode.
					// Critical to do it here and consider wait time.
					// Starvation mode is so inefficient, that two goroutines
					// can go lock-step infinitely once they switch mutex
					// to starvation mode.
					delta -= mutexStarving
				}
				atomic.AddInt32(&m.state, delta)
				break
			}
			awoke = true
			iter = 0
		} else {
			old = m.state
		}
	}
}
```

其中 `runtime_doSpin` 会调用 `procyield` 汇编方法，执行 30 次 PAUSE 指令占用 CPU 时间。

```asm
TEXT runtime·procyield(SB),NOSPLIT,$0-0
	MOVL	cycles+0(FP), AX
again:
	PAUSE
	SUBL	$1, AX
	JNZ	again
	RET
```

若是长时间未获取到锁，就进入互斥锁的第二阶段，使用信号量进行同步。

##### 互斥锁的释放

互斥锁的释放与互斥锁的获取相对应。如果当前锁处于普通的锁定状态，则直接修改状态后退出。如果当前锁处于饥饿状态，则进入信号量同步阶段，到全局哈希表中寻找当前锁的等待队列，以先入先出的顺序唤醒指定协程。



#### 读写锁

读写锁位于 `sync` 标准库中，其复用了互斥锁及信号量这两种机制。

```go
// A RWMutex is a reader/writer mutual exclusion lock.
// The lock can be held by an arbitrary number of readers or a single writer.
// The zero value for a RWMutex is an unlocked mutex.
//
// A RWMutex must not be copied after first use.
//
// If a goroutine holds a RWMutex for reading and another goroutine might
// call Lock, no goroutine should expect to be able to acquire a read lock
// until the initial read lock is released. In particular, this prohibits
// recursive read locking. This is to ensure that the lock eventually becomes
// available; a blocked Lock call excludes new readers from acquiring the
// lock.
//
// In the terminology of the Go memory model,
// the n'th call to Unlock “synchronizes before” the m'th call to Lock
// for any n < m, just as for Mutex.
// For any call to RLock, there exists an n such that
// the n'th call to Unlock “synchronizes before” that call to RLock,
// and the corresponding call to RUnlock “synchronizes before”
// the n+1'th call to Lock.
type RWMutex struct {
	w           Mutex  // held if there are pending writers
	writerSem   uint32 // semaphore for writers to wait for completing readers
	readerSem   uint32 // semaphore for readers to wait for completing writers
	readerCount int32  // number of pending readers
	readerWait  int32  // number of departing readers
}
```



