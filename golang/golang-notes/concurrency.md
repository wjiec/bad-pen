并发
-------

并发：逻辑上具备同时处理多个任务的能力

并行：物理上在同一时刻执行多个并发任务

> Concurrency is not parallelism: different concurrent design enable different ways to parallelize.

通常情况下，用多进程来实现分布式和负载均衡，减轻单进程垃圾回收压力；用多线程抢夺更多的处理器资源；用协程来提高处理器时间片利用率。



### 通道

Go鼓励使用 CSP 通道，以通信来代替内存共享，实现并发安全。

> Don't communicate by sharing memory, share memory by communicating.
>
> CSP: Communicating Sequential Process.



小技巧：如果要等全部通道消息处理完毕，可将已完成通道设置为 nil，这样它就就会被阻塞，从而不再被 select 选中

```go
func Print(a, b, c, d <-chan int) {
	for {
		select {
		case v, ok := <-a:
			if ok {
				fmt.Printf("a: %d\n", v)
			} else {
				a = nil
			}
		case v, ok := <-b:
			if ok {
				fmt.Printf("b: %d\n", v)
			} else {
				b = nil
			}
		case v, ok := <-c:
			if ok {
				fmt.Printf("c: %d\n", v)
			} else {
				c = nil
			}
		case v, ok := <-d:
			if ok {
				fmt.Printf("d: %d\n", v)
			} else {
				d = nil
			}
		default:
			return
		}
	}
}

func main() {
	g := func(n int) chan int {
		ch := make(chan int, 10)
		for i := 0; i < n; i++ {
			ch <- rand.Int()
		}
		close(ch)
		return ch
	}

	Print(g(1), g(2), g(3), g(4))
}
```



性能：将发往通道的数据打包，减少传输次数，可有效提升性能。从实现上来说，通道队列依旧使用锁同步机制，单词获取更多的数据（批处理），可改善因频繁加锁造成的性能问题。



资源泄露：通道可能会引发 goroutine leak，准确的说，是指goroutine 处于发送或接收的阻塞状态，但一直未被唤醒。垃圾回收器并不收集此类资源，导致它们会在等待队列里长久休眠，形成资源泄露。



小知识：如果 channel 未关闭，将会被垃圾处理器回收

```go
type ChannelHolder struct {
	c chan int
	v int
}

func MakeChannel() {
	ch := &ChannelHolder{
		c: make(chan int),
		v: 1,
	}

	if false {
		fmt.Println(ch.v)
	}
}

// GODEBUG=gctrace=1
func main() {
	go func() {
		for {
			for i := 0; i < 4096; i++ {
				MakeChannel()
			}
			fmt.Println("make channels ...")
			time.Sleep(500 * time.Millisecond)
		}
	}()

	for {
		time.Sleep(10 * time.Second)
		runtime.GC()
	}
}
```

