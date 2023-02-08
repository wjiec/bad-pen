深入垃圾回收全流程
----------------------------

垃圾回收贯穿于程序的整个生命周期，运行时将循环不断地检测当前程序的内存使用状态并选择在核实的时机执行垃圾回收。



### 垃圾回收循环

当内存达到了垃圾回收的阈值时，将触发新一轮的垃圾回收。之后会先后经历标记准备阶段、并行标记阶段、标记终止阶段及垃圾清扫阶段。在并行标记阶段引入了辅助标记技术，在垃圾清扫阶段还引入了辅助清扫、系统驻留内存清除技术。



### 标记准备阶段

标记准备阶段最重要的任务是清扫上一阶段 GC 遗留的需要清理的对象，因为使用了懒清扫算法，所以当执行下一次 GC 时，可能还有垃圾对象没有被清扫。

标记准备阶段会为每个逻辑处理器 P 启动一个标记协程，但并不是所有的标记协程都有执行的机会。在这个阶段需要解决两个重要的问题：

* 如何决定需要多少标记协程
* 如何调度标记协程

#### 计算标记协程的数量

在标记准备阶段，会计算当前后台需要开启多少标记协程。目前 Go 语言规定后台标记协程消耗 的 CPU 应该接近 25%。其核心逻辑位于 `startCycle` 函数中：

```go
//
// runtime/mgcpacer.go
//
// startCycle resets the GC controller's state and computes estimates
// for a new GC cycle. The caller must hold worldsema and the world
// must be stopped.
func (c *gcControllerState) startCycle(markStartTime int64, procs int, trigger gcTrigger) {
	// ...

	// Compute the background mark utilization goal. In general,
	// this may not come out exactly. We round the number of
	// dedicated workers so that the utilization is closest to
	// 25%. For small GOMAXPROCS, this would introduce too much
	// error, so we add fractional workers in that case.
	totalUtilizationGoal := float64(procs) * gcBackgroundUtilization
	c.dedicatedMarkWorkersNeeded = int64(totalUtilizationGoal + 0.5)
	utilError := float64(c.dedicatedMarkWorkersNeeded)/totalUtilizationGoal - 1
	const maxUtilError = 0.3
	if utilError < -maxUtilError || utilError > maxUtilError {
		// Rounding put us more than 30% off our goal. With
		// gcBackgroundUtilization of 25%, this happens for
		// GOMAXPROCS<=3 or GOMAXPROCS=6. Enable fractional
		// workers to compensate.
		if float64(c.dedicatedMarkWorkersNeeded) > totalUtilizationGoal {
			// Too many dedicated workers.
			c.dedicatedMarkWorkersNeeded--
		}
		c.fractionalUtilizationGoal = (totalUtilizationGoal - float64(c.dedicatedMarkWorkersNeeded)) / float64(procs)
	} else {
		c.fractionalUtilizationGoal = 0
	}

	// ...
}
```

#### 切换到后台标记协程

每个逻辑处理器 P 进入新一轮的调度循环时，调度器会判断程序是否处于 GC 阶段，如果是，则尝试判断当前 P 是否需要执行后台标记任务



### 并发标记阶段

在并发标记阶段，后台标记协程可以与执行用户代码的协程并行。

#### 根对象扫描

扫描的第一阶段是扫描根对象。根对象是最基本的对象，从根对象出发，可以找到所有的引用对象（即活着的对象）。在 Go 语言中，根对象包括全局变量（在 .bss 和 .data 段内存中）、span 中 finalizer 的任务，以及所有的协程栈。

* 全局变量扫描：这需要编译时与运行时的共同努力。
* finalizer：这是一个特殊的对象，其是在对象释放后会被调用的析构器，用于资源释放。
* 栈扫描：栈扫描是根对象扫描中最重要的部分，栈扫描需要编译时与运行时的共同努力，运行时能够计算出当前协程栈的所有栈帧信息，而编译时能够得知栈上有哪些指针，以及对象中的那一部分包含了指针。

#### 扫描灰色对象

全局变量、析构器、所有协程的栈都会被扫描，从而标记目前还在使用的内存对象。下一个是从这些被标记为灰色的内存对象出发，进一步标记整个堆内存中活着的对象。

在标记期间、会循环往复地从本地标记队列获取灰色对象，从灰色对象扫描到的白色对象仍然会放入标记队列中，如果扫描到以及被标记的对象则忽略，一直到队伍中的任务为空为止。



### 标记终止阶段

标记终止阶段主要完成一些指标，例如统计用时、统计强制开始 GC 的次数、更新下一次触发 GC 需要达到的堆目标、关闭写屏障等，并唤醒后台清扫的协程，开始下一阶段的清扫工作。

标记终止阶段的重要任务是计算下一次触发 GC 时需要达成的堆目标，这叫做垃圾回收的调步算法。GC 过程中的偏差率公式为：

```
偏差率 = (目标增长率 - 触发率) - 调整率 * (实际增长率 - 触发率)
```

其中 `调整率 = GC 标记阶段的 CPU 使用率 / 目标 CPU 占用率 `。



### 辅助标记

引入并发标记之后，如果用户分配内存的速度大于后台标记的速度，那么 GC 标记将永远不会结束，从而无法完成完整的 GC 周期，造成内存泄露。所以为了解决该问题，引入辅助标记算法（辅助标记算法必须在垃圾回收的标记阶段运行）。

所谓辅助标记算法，即定义用户分配的内存为 M，而扫描的速度为 X，则需要要求 X >= M。而具体的实现方式是在 GC 并发标记阶段，当用户协程分配内存时，会先检查是否已经完成了指定的扫描工作。

**用户协程中的本地资产来于后台标记协程的扫描工作，如果工作协程在分配内存时，无法从本地资产和全局资产池中获取资产，那么就需要停止工作，并执行辅助标记协程。**



### 屏障技术

屏障技术解决的是更为棘手的问题——准确性。保证并发标记准确性需要遵守强、弱三色不变性。**强三色不变性指所有白色对象都不能被黑色对象引用**，**弱三色不变性允许白色对象被黑色对象引用，但是白色对象必须有一条路径始终是被灰色对象引用的**，这保证了该对象最终能被扫描到。

**屏障技术的原则是在写入或者删除对象时将可能活着的对象标记为灰色。**

在 Go 语言中，混合写屏障技术的实现依赖编译时与运行时的共同努力，在标记准备阶段的 STW 阶段会打开写屏障，具体做法是讲全局变量 writeBarrier.enable 设置为 true。

编译器会在所有堆写入或删除操作前判断当前是否为垃圾回收标记阶段，如果是则会执行对应的混合写屏障标记对象。



### 垃圾清扫

在标记结束之后会进入垃圾清扫阶段，将垃圾对象的内存回收重用或返还给操作系统。在清扫阶段会调用 gcSweep 函数，该函数会将 sweep.g 清扫协程的状态变为 running，在结束 STW 阶段并开始重新调度循环时优先调度清扫协程。

#### 懒清扫逻辑

清扫是以 span 为单位进行的，sweepone 函数的作用是找到一个 span 并进行相应的清扫工作。需要清扫的 span 队列是一个长度为 2 的队列，每次都将互换切换着进行清扫。

#### 辅助清扫

辅助清扫即工作协程必须在适当的时机执行辅助清扫工作，以避免下一次 GC 发生时还有大量的未清扫 span。



### 系统驻留内存清除

为了将系统分配的内存保持在适当的大小，同时回收不再被使用的内存，Go 语言使用了单独的后台清扫协程来清除内存。

```go
// gcenable is called after the bulk of the runtime initialization,
// just before we're about to start letting user code run.
// It kicks off the background sweeper goroutine, the background
// scavenger goroutine, and enables GC.
func gcenable() {
	// Kick off sweeping and scavenging.
	c := make(chan int, 2)
	go bgsweep(c)
	go bgscavenge(c)
	<-c
	<-c
	memstats.enablegc = true // now that runtime is initialized, GC is okay
}
```



### 通过 trace 工具来检查垃圾回收产生的性能问题

我们可以通过 trace 工具来分析内存在某段时间内的增长情况：

```shell
curl -o trace.out http://localhost:8088/debug/pprof/trace?seconds=30

go tool trace trace.out
```

