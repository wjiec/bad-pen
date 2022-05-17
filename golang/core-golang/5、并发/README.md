并发
------

现有的软件对并发的支持不是很友好，而Go语言就是在这个背景下为解决并发编程而诞生。



### 并发基础

1、并行意味着程序在**任意时刻**都是同时运行的。说明的是程序任一粒度的时间内都具备同时执行的能力，最简单的并行就是多机。

2、并发意味着程序在**单位时间**内是同时运行的。并发强调在规定的时间内多个请求都能得到执行和处理，*给外部一种同时执行的感觉*。实际上内部可能是分时操作的。并发重在避免阻塞，使程序不会因为一个阻塞而停止处理。分时操作系统就是并发的典型应用场景。

并行是硬件和操作系统开发者重点考虑的问题，而应用层的程序员需要结合实际需求设计出具有良好并发结构的程序，提升程序的并发处理能力。

#### Goroutine

Go语言通过在用户层上再构造一级调度，将并发的粒度进一步降低（进程和线程的切换需要保存现场状态切换上下文），达到更大限度地提升程序运行效率。在Go语言中的并发执行体称为 goroutine，可以通过 `go` 关键字来启动一个 goroutine。如下：

```go
go func() {
    for i := 0; i < 1000; i++ {
        fmt.Println("hello, world!")
    }
}()
```

goroutine 有以下这些特性：

* go 所创建的 goroutine 是非阻塞的，不会等待函数完成
* go 所调用的函数的返回值会被忽略
* 调度器不能保证多个 goroutine 的执行次序
* 没有父子 goroutine 的概念，所有的 goroutine 都是平等地调度和执行
* Go 程序在执行时会单独为 main 函数创建一个 goroutine
* Go 没有没有暴露 goroutine id 给用户，所以不能在一个 goroutine 里面显式地操作另一个 goroutine。不过可以通过 runtime 包中提供的其他函数访问和设置 goroutine 的相关信息。

#### runtime包中和goroutine有关的函数

1、`runtime.GOMAXPROCS`：该函数设置并返回之前可以并发执行的 goroutine 数量（如果为0则不修改，表现的语义是查询当前的可以并发执行的 goroutine 数量）

```go
func main() {
    fmt.Println("GOMAXPROCS = ", runtime.GOMAXPROCS(0))

    runtime.GOMAXPROCS(2)
    fmt.Println("GOMAXPROCS = ", runtime.GOMAXPROCS(0))
    
    // Output:
    //  GOMAXPROCS =  8
    //  GOMAXPROCS =  2
}
```

2、`runtime.Goexit`：结束当前调用该方法的 goroutine。在结束之前会调用已经注册的 defer 方法。同时 `runtime.Goexit` 不会导致 panic，所以 defer 中的 recover 方法会返回 nil。

3、`runtime.Gosched`：放弃当前 goroutine 获取的时间片，并在等待队列中等待下次调度。

#### 通信：chan

通道 channel 是 goroutine 之间通信和同步的重要组件。Go 语言的哲学是“不要通过共享内存来通信，而是通过通信来共享内存”。在 Go 语言中通道是有类型的，未初始化的同党没有任何意义，其值是 nil。我们可以通过 `make` 方法来初始化一个无缓冲或有缓存的通道。我们可以通过 `len` 和 `cap` 方法来获取通道中未读取的数据数和通道的容量（无缓存的通道的 len 和 cap 都是 0）。通道的使用一般有以下几种方式

1、通过无缓存的通道实现 goroutine 之前的同步等待

```go
func main() {
    c := make(chan struct{})

    go func() {
        var sum int
        for i := 0; i < 1000; i++ {
            sum += i
            time.Sleep(time.Millisecond)
        }

        fmt.Printf("Sum = %d\n", sum)
        c <- struct{}{}
        close(c)
    }()

    <-c
    fmt.Println("Goroutine terminated ...")
}
```

2、在关闭通道后，已写入缓冲通道的数据不会消息，它可以缓冲和适配两个 goroutine 处理速率不一致的情况，有削峰填谷和增大吞吐量的功能

```go
func main() {
    c := make(chan int, 888)

    go func() {
        for i := 0; i < 1000; i++ {
            c <- i
        }

        close(c)
        log.Println("Provides 1000 numbers ...")
    }()

    stop := make(chan struct{})
    go func() {
        var sum int
        for v := range c {
            sum += v
            time.Sleep(time.Millisecond)
        }
        log.Printf("Sum = %d\n", sum)
        stop <- struct{}{}
        close(stop)
    }()

    <-stop
    
    // Output:
    //  2022/05/16 23:58:13 Provides 1000 numbers ...
    //  2022/05/16 23:58:14 Sum = 499500
}
```

##### 操作不同状态的 chan 可能出现的情况

* panic
  * 向已经关闭的通道写数据会导致 panic
  * 重复关闭通道会导致 panic
* 阻塞
  * 向未初始化的通道读数据或写数据会导致 goroutine 永久阻塞
  * 向缓冲区已满的通道写数据会导致 goroutine 阻塞
  * 读取没有缓冲数据的通道会导致 goroutine 阻塞
* 非阻塞
  * 读取已经关闭的通道不会阻塞，而是直接返回通道元素类型的零值，可以使用 `comma, ok` 语法来检查通道是否关闭
  * 向有缓冲且没有满的通道读写数据不会导致阻塞

#### WaitGroup

`sync` 包中提供了多个 goroutine 同步的机制，主要是用过 `WaitGroup` 来实现的

```go
func main() {
    urls := []string{
        "https://www.github.com",
        "https://www.stackoverflow.com",
        "https://news.ycombinator.com",
    }

    var wg sync.WaitGroup
    for _, url := range urls {
        wg.Add(1)
        go func(url string) {
            defer wg.Done()

            resp, err := http.Get(url)
            if err != nil {
                log.Printf("Request error: %s\n", err)
                return
            }
            defer func() { _ = resp.Body.Close() }()

            log.Printf("Request %q: %s\n", url, resp.Status)
        }(url)
    }

    wg.Wait()

    // Output:
    //    2022/05/17 00:09:48 Request "https://news.ycombinator.com": 200 OK
    //    2022/05/17 00:09:48 Request "https://www.github.com": 200 OK
    //    2022/05/17 00:09:49 Request "https://www.stackoverflow.com": 200 OK
}
```

#### select

`select` 是类 UNIX 系统中提供用于多路复用的系统 API。Go语言借助其多路复用的概念，提供了 `select` 关键字，同于同时监听多个通道。

1、当没有通道可读/可写时，select 是阻塞的

2、只要监听的通道有一个是可读或可写的，则会直接进入就绪通道的处理逻辑

3、如果有多个通道可读/可写，则 select 随机选择一个镜像处理

```go
func main() {
    ci := make(chan int)
    cc := make(chan struct{})
    close(cc)

    go func() {
        for {
            select {
            case ci <- 0:
            case ci <- 1:
            case <-cc:
                log.Println("closed channel")
            }
        }
    }()

    for i := 0; i < 3; i++ {
        log.Println(<-ci)
    }

    // Output:
    //  2022/05/17 00:19:29 closed channel
    //  2022/05/17 00:19:29 closed channel
    //  2022/05/17 00:19:29 0
    //  2022/05/17 00:19:29 closed channel
    //  2022/05/17 00:19:29 closed channel
    //  2022/05/17 00:19:29 closed channel
    //  2022/05/17 00:19:29 closed channel
    //  2022/05/17 00:19:29 closed channel
    //  2022/05/17 00:19:29 closed channel
    //  2022/05/17 00:19:29 closed channel
    //  2022/05/17 00:19:29 0
    //  2022/05/17 00:19:29 closed channel
    //  2022/05/17 00:19:29 1
}
```

#### 扇入（Fan in）和扇出（Fan out）

所谓扇入就是将多路通道聚合到一条通道中处理，在 Go 中最简单的扇入就是使用 select 聚合多条通道。当生产者的速度很慢时，需要使用扇入来聚合多个生产者以满足消费者。

扇出就是将一条通道发散到多条通道中处理，在Go语言中就是使用 go 关键字启动多个 goroutine 并发处理。当消费者速度很慢时，就需要使用扇出技术来并发处理任务。

#### 通知退出机制

select 能感知到所监听的某个通道被关闭了，然后进行相应的处理，由此我们可以实现“通知退出机制”（close channel to broadcast）

```go
func Generator(done <-chan struct{}) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)

		for {
			select {
			case <-done:
				return
			case ch <- rand.Int():
			}
		}
	}()
	return ch
}

func main() {
	done := make(chan struct{})
	g := Generator(done)

	log.Println(<-g)
	log.Println(<-g)

	close(done)

	log.Println(<-g)
	log.Println(<-g)

	// Output:
	//  2022/05/17 00:39:31 5577006791947779410
	//  2022/05/17 00:39:31 8674665223082153551
	//  2022/05/17 00:39:31 6129484611666145821
	//  2022/05/17 00:39:31 0

	// 这里在 close 之后有2个操作在同步进行
	//	1. 尝试从 g 中读取一个随机数
	//	2. select 检查是否有哪个通道准备就绪
	//
	// 如果此时 select 选择 (1) 则可以继续执行后续的输出随机数
	// 如果次数 select 选择 (2) 则后续输出零值

	// Output:
	//  2022/05/17 00:39:48 5577006791947779410
	//  2022/05/17 00:39:48 8674665223082153551
	//  2022/05/17 00:39:48 0
	//  2022/05/17 00:39:48 0
}
```



### 并发范式

1、在应用中，比较常见的场景就是调用一个统一的全局生成器服务，用于生成订单号、序列号和随机数等。

```go
func Generate(done <-chan struct{}) <-chan int {
	ch := make(chan int, 100)

	go func() {
		defer close(ch)

		for {
			select {
			case <-done:
				return
			case ch <- rand.Int():
			}
		}
	}()
	return ch
}

func Generator(done <-chan struct{}) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)

		a := Generate(done)
		b := Generate(done)

		for {
			select {
			case ch <- <-a: // 扇入
			case ch <- <-b: // 扇入
			case <-done:
				return
			}
		}
	}()
	return ch
}

func main() {
	done := make(chan struct{})

	g := Generator(done)
	for i := 0; i < 10; i++ {
		fmt.Println("Generate", <-g)
	}

	close(done)
}
```

2、具有多个相同管道类型参数的函数可以组合成一个调用链，形成一个类管道操作

```go
func Multiply(ns <-chan int, val int) <-chan int {
	ch := make(chan int)
	go func() {
		for n := range ns {
			ch <- n * val
		}

		close(ch)
	}()
	return ch
}

func main() {
	ch := make(chan int)
	go func() {
		for i := 0; i < 100; i++ {
			ch <- i
		}

		close(ch)
	}()

	g2 := Multiply(ch, 2)
	g6 := Multiply(g2, 3)
	g48 := Multiply(g6, 8)

	for v := range g48 {
		fmt.Println(v)
	}
}
```

3、每个请求使用一个 goroutine 进行处理

```go
//
// net/http/server.go
//

// Serve accepts incoming connections on the Listener l, creating a
// new service goroutine for each. The service goroutines read requests and
// then call srv.Handler to reply to them.
//
// HTTP/2 support is only enabled if the Listener returns *tls.Conn
// connections and they were configured with "h2" in the TLS
// Config.NextProtos.
//
// Serve always returns a non-nil error and closes l.
// After Shutdown or Close, the returned error is ErrServerClosed.
func (srv *Server) Serve(l net.Listener) error {
	if fn := testHookServerServe; fn != nil {
		fn(srv, l) // call hook with unwrapped listener
	}

	origListener := l
	l = &onceCloseListener{Listener: l}
	defer l.Close()

	if err := srv.setupHTTP2_Serve(); err != nil {
		return err
	}

	if !srv.trackListener(&l, true) {
		return ErrServerClosed
	}
	defer srv.trackListener(&l, false)

	baseCtx := context.Background()
	if srv.BaseContext != nil {
		baseCtx = srv.BaseContext(origListener)
		if baseCtx == nil {
			panic("BaseContext returned a nil context")
		}
	}

	var tempDelay time.Duration // how long to sleep on accept failure

	ctx := context.WithValue(baseCtx, ServerContextKey, srv)
	for {
		rw, err := l.Accept()
		if err != nil {
			select {
			case <-srv.getDoneChan():
				return ErrServerClosed
			default:
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				srv.logf("http: Accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		connCtx := ctx
		if cc := srv.ConnContext; cc != nil {
			connCtx = cc(connCtx, rw)
			if connCtx == nil {
				panic("ConnContext returned nil")
			}
		}
		tempDelay = 0
		c := srv.newConn(rw)
		c.setState(c.rwc, StateNew, runHooks) // before Serve can return
		go c.serve(connCtx) // 这里对每个连接都会新开一个 goroutine 进行处理
	}
}
```

4、固定数量的 goroutine 池

```go
type Task struct{}

func Work(task *Task) {
	// do work
}

func Start(tasks <-chan *Task, n int) {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for t := range tasks {
				Work(t)
			}
		}()
	}
	wg.Wait()
}

func main() {
	ch := make(chan *Task)
	Start(ch, 10)
}
```

### Context标准库

