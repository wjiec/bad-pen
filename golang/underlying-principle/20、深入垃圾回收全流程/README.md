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



