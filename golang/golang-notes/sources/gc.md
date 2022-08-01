垃圾回收
------------

Go 语言中 GC 的基本特征是“非分代、非紧缩、写屏障、并发标记清理”。在 Go 源码中有 GC 的详细说明：

```go
//
// runtime/mgc.go
//

// Garbage collector (GC).
//
// The GC runs concurrently with mutator threads, is type accurate (aka precise), allows multiple
// GC thread to run in parallel. It is a concurrent mark and sweep that uses a write barrier. It is
// non-generational and non-compacting. Allocation is done using size segregated per P allocation
// areas to minimize fragmentation while eliminating locks in the common case.
```

#### 三色标记和写屏障

为了让标记和用户代码能并发运行的基本原理（DFS）是：

* 起初所有对象都是白色
* 扫描找出所有可达对象，标记为灰色，放入待处理队列（入度为 1）、
* 从待处理队列中获取一个灰色对象，将其所引用的其他对象标记为灰色后放入队列，自身则变为黑色
* 写屏障监视对象内存修改，重新标记或返回队列

当完成所有的扫描和标记工作之后，剩余的对象为白色的就是待回收对象，而黑色则是活跃对象。



#### 控制器

控制器全程参与并发回收任务，记录相关状态数据，动态调整运行策略，影响并发标记单元的工作模式和数量，平衡CPU资源占用。在回收结束之后，参与 next_gc 回收阈值设置，调整垃圾回收频率。

```go
// gcController implements the GC pacing controller that determines
// when to trigger concurrent garbage collection and how much marking
// work to do in mutator assists and background marking.
//
// It uses a feedback control algorithm to adjust the gcController.trigger
// trigger based on the heap growth and GC CPU utilization each cycle.
// This algorithm optimizes for heap growth to match GOGC and for CPU
// utilization between assist and background marking to be 25% of
// GOMAXPROCS. The high-level design of this algorithm is documented
// at https://golang.org/s/go15gcpacing.
//
// All fields of gcController are used only during a single mark
// cycle.
var gcController gcControllerState
```



#### 辅助回收

某些时候，对象分配速度可能远快于后台标记速度，这可能会导致堆无限扩张甚至让垃圾回收永远无法完成。此时让用户代码线程参与后台回收标记就非常有必须奥。在为对象分配堆内存时，通过相关策略去执行一定限度的回收操作，平衡分配和回收操作，让进程处于良性状态。



### 初始化

GC 的初始化过程非常简单：

```go
func gcinit() {
	if unsafe.Sizeof(workbuf{}) != _WorkbufSize {
		throw("size of Workbuf is suboptimal")
	}
	// No sweep on the first cycle.
	mheap_.sweepDrained = 1

	// Initialize GC pacer state.
	// Use the environment variable GOGC for the initial gcPercent value.
	gcController.init(readGOGC())

	work.startSema = 1
	work.markDoneSema = 1
	lockInit(&work.sweepWaiters.lock, lockRankSweepWaiters)
	lockInit(&work.assistQueue.lock, lockRankAssistQueue)
	lockInit(&work.wbufSpans.lock, lockRankWbufSpans)
}

func readGOGC() int32 {
	p := gogetenv("GOGC")
	if p == "off" {
		return -1
	}
	if n, ok := atoi32(p); ok {
		return n
	}
	return 100
}

func (c *gcControllerState) init(gcPercent int32) {
	c.heapMinimum = defaultHeapMinimum

	// Set a reasonable initial GC trigger.
	c.triggerRatio = 7 / 8.0

	// Fake a heapMarked value so it looks like a trigger at
	// heapMinimum is the appropriate growth from heapMarked.
	// This will go into computing the initial GC goal.
	c.heapMarked = uint64(float64(c.heapMinimum) / (1 + c.triggerRatio))

	// This will also compute and set the GC trigger and goal.
	c.setGCPercent(gcPercent)
}

// setGCPercent updates gcPercent and all related pacer state.
// Returns the old value of gcPercent.
//
// The world must be stopped, or mheap_.lock must be held.
func (c *gcControllerState) setGCPercent(in int32) int32 {
	assertWorldStoppedOrLockHeld(&mheap_.lock)

	out := c.gcPercent
	if in < 0 {
		in = -1
	}
	c.gcPercent = in
	c.heapMinimum = defaultHeapMinimum * uint64(c.gcPercent) / 100
	// Update pacing in response to gcPercent change.
	c.commit(c.triggerRatio)

	return out
}
```



### 启动

在为对象分配堆内存后，`mallocgc` 函数会检查垃圾回收触发条件，并依照相关状态启动或参与辅助回收

```go
//
// runtime/mallocgc.go
//


// Allocate an object of size bytes.
// Small objects are allocated from the per-P cache's free lists.
// Large objects (> 32 kB) are allocated straight from the heap.
func mallocgc(size uintptr, typ *_type, needzero bool) unsafe.Pointer {
	if gcphase == _GCmarktermination {
		throw("mallocgc called with gcphase == _GCmarktermination")
	}
    
    // assistG is the G to charge for this allocation, or nil if
	// GC is not currently active.
	var assistG *g
	if gcBlackenEnabled != 0 {
		// Charge the current user G for this allocation.
		assistG = getg()
		if assistG.m.curg != nil {
			assistG = assistG.m.curg
		}
		// Charge the allocation against the G. We'll account
		// for internal fragmentation at the end of mallocgc.
		assistG.gcAssistBytes -= int64(size)

		if assistG.gcAssistBytes < 0 {
            // 辅助垃圾回收
			// This G is in debt. Assist the GC to correct
			// this before allocating. This must happen
			// before disabling preemption.
			gcAssistAlloc(assistG)
		}
	}

	// Allocate black during GC.
	// All slots hold nil so no scanning is needed.
	// This may be racing with GC so do it atomically if there can be
	// a race marking the bit.
	if gcphase != _GCoff {
        // 直接分配黑色标记对象
		gcmarknewobject(span, uintptr(x), size, scanSize)
	}

    // 启动并发垃圾回收
	if shouldhelpgc {
		if t := (gcTrigger{kind: gcTriggerHeap}); t.test() {
			gcStart(t)
		}
	}

	return x
}
```

垃圾回收默认以全并发模式运行，但可以用环境变量或参数禁用并发标记和并发清理。 GC goroutine 一直循环直到符合条件时被唤醒。

```go
//
// runtime/mgc.go
//

// gcStart starts the GC. It transitions from _GCoff to _GCmark (if
// debug.gcstoptheworld == 0) or performs all of GC (if
// debug.gcstoptheworld != 0).
//
// This may return without performing this transition in some cases,
// such as when called on a system stack or with locks held.
func gcStart(trigger gcTrigger) {
    // 并发标记模式
	// In gcstoptheworld debug mode, upgrade the mode accordingly.
	// We do this after re-checking the transition condition so
	// that multiple goroutines that detect the heap trigger don't
	// start multiple STW GCs.
	mode := gcBackgroundMode
	if debug.gcstoptheworld == 1 {
		mode = gcForceMode
	} else if debug.gcstoptheworld == 2 {
		mode = gcForceBlockMode
	}

	gcBgMarkStartWorkers()
	systemstack(gcResetMarkState)

    // 同步阻塞模式
	// In STW mode, disable scheduling of user Gs. This may also
	// disable scheduling of this goroutine, so it may block as
	// soon as we start the world again.
	if mode != gcBackgroundMode {
		schedEnableUser(false)
	}

	// Enter concurrent mark phase and enable
	// write barriers.
	//
	// Because the world is stopped, all Ps will
	// observe that write barriers are enabled by
	// the time we start the world and begin
	// scanning.
	//
	// Write barriers must be enabled before assists are
	// enabled because they must be enabled before
	// any non-leaf heap objects are marked. Since
	// allocations are blocked until assists can
	// happen, we want enable assists as early as
	// possible.
	setGCPhase(_GCmark)

	gcBgMarkPrepare() // Must happen before assist enable.
	gcMarkRootPrepare()

	// Mark all active tinyalloc blocks. Since we're
	// allocating from these, they need to be black like
	// other allocations. The alternative is to blacken
	// the tiny block on every allocation from it, which
	// would slow down the tiny allocator.
	gcMarkTinyAllocs()

	// At this point all Ps have enabled the write
	// barrier, thus maintaining the no white to
	// black invariant. Enable mutator assists to
	// put back-pressure on fast allocating
	// mutators.
	atomic.Store(&gcBlackenEnabled, 1)

	// Assists and workers can start the moment we start
	// the world.
	gcController.markStartTime = now

	// In STW mode, we could block the instant systemstack
	// returns, so make sure we're not preemptible.
	mp = acquirem()

	// Concurrent mark.
	systemstack(func() {
		now = startTheWorldWithSema(trace.enabled)
		work.pauseNS += now - work.pauseStart
		work.tMark = now
		memstats.gcPauseDist.record(now - work.pauseStart)
	})

	// Release the world sema before Gosched() in STW mode
	// because we will need to reacquire it later but before
	// this goroutine becomes runnable again, and we could
	// self-deadlock otherwise.
	semrelease(&worldsema)
	releasem(mp)

	// Make sure we block instead of returning to user code
	// in STW mode.
	if mode != gcBackgroundMode {
		Gosched()
	}

	semrelease(&work.startSema)
}
```

整个并发回收过程位于 `runtime.GC` 函数中：

```go
// GC runs a garbage collection and blocks the caller until the
// garbage collection is complete. It may also block the entire
// program.
func GC() {
	// We consider a cycle to be: sweep termination, mark, mark
	// termination, and sweep. This function shouldn't return
	// until a full cycle has been completed, from beginning to
	// end. Hence, we always want to finish up the current cycle
	// and start a new one. That means:
	//
	// 1. In sweep termination, mark, or mark termination of cycle
	// N, wait until mark termination N completes and transitions
	// to sweep N.
	//
	// 2. In sweep N, help with sweep N.
	//
	// At this point we can begin a full cycle N+1.
	//
	// 3. Trigger cycle N+1 by starting sweep termination N+1.
	//
	// 4. Wait for mark termination N+1 to complete.
	//
	// 5. Help with sweep N+1 until it's done.
	//
	// This all has to be written to deal with the fact that the
	// GC may move ahead on its own. For example, when we block
	// until mark termination N, we may wake up in cycle N+2.

	// Wait until the current sweep termination, mark, and mark
	// termination complete.
	n := atomic.Load(&work.cycles)
	gcWaitOnMark(n)

	// We're now in sweep N or later. Trigger GC cycle N+1, which
	// will first finish sweep N if necessary and then enter sweep
	// termination N+1.
	gcStart(gcTrigger{kind: gcTriggerCycle, n: n + 1})

	// Wait for mark termination N+1 to complete.
	gcWaitOnMark(n + 1)

	// Finish sweep N+1 before returning. We do this both to
	// complete the cycle and because runtime.GC() is often used
	// as part of tests and benchmarks to get the system into a
	// relatively stable and isolated state.
	for atomic.Load(&work.cycles) == n+1 && sweepone() != ^uintptr(0) {
		sweep.nbgsweep++
		Gosched()
	}

	// Callers may assume that the heap profile reflects the
	// just-completed cycle when this returns (historically this
	// happened because this was a STW GC), but right now the
	// profile still reflects mark termination N, not N+1.
	//
	// As soon as all of the sweep frees from cycle N+1 are done,
	// we can go ahead and publish the heap profile.
	//
	// First, wait for sweeping to finish. (We know there are no
	// more spans on the sweep queue, but we may be concurrently
	// sweeping spans, so we have to wait.)
	for atomic.Load(&work.cycles) == n+1 && !isSweepDone() {
		Gosched()
	}

	// Now we're really done with sweeping, so we can publish the
	// stable heap profile. Only do this if we haven't already hit
	// another mark termination.
	mp := acquirem()
	cycle := atomic.Load(&work.cycles)
	if cycle == n+1 || (gcphase == _GCmark && cycle == n+2) {
		mProf_PostSweep()
	}
	releasem(mp)
}
```



### 标记

并发标记分为两个步骤：

* 扫描：遍历相关内存区域，依照指针标记找出灰色可达对象，加入队列
* 标记：将灰色对象从队列中取出，将其引用对象标记为灰色，自身标记为黑色



### 清理

清理操作要简单的多，此时所有未被标记的白色对象都不再被引用，可简单地将其内存回收。



### 监控

当瞬间分配大量对象时，可能会将垃圾回收的触发条件 `next_gc` 推到一个很大的值，当活跃对象远小于该阈值时，造成垃圾回收久久无法触发。同样的情况也可能是因为某个算法在短期内大量使用临时对象造成。

监控服务 sysmon 是垃圾回收期的最后一道保险措施，该服务每 2 分钟就会检查一次垃圾回收状态，如超出 2 分钟未触发，那就强制执行。
