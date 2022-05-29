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

context 库的设计目的就是跟踪 goroutine 调用树，并在这些 goroutine 调用树中传递通知和元数据：

* 退出通知机制：通知可以传递给整个 goroutine 调用树上的每一个 goroutine
* 传递数据：数据可以传递给整个 goroutine 调用树上的每一个 goroutine

#### Context 标准库中的 API 函数

1、首先是用于构造 Context 树的根节点对象，一般作为后续 `With*` 包装方法的实参

```go
func Background() Context
func TODO() Context
```

2、`With*` 包装方法用来构建具有不同功能的 Context 对象

```go
// 创建带有取消功能的上下文对象
func WithCancel(parent Context) (ctx Context, cancel CancelFUnc)

// 创建带有超时取消功能的上下文对象
func WithDeadline(parent Context, deadline time.Time) (ctx Context, cancel CancelFUnc)
func WithTimeout(parent Context, timeout time.Duration) (ctx Context, cancel CancelFUnc)

// 创建一个能传递数据的上下文对象
func WithValue(parent Context, key, val interface{}) Context
```

#### 使用 Context 传递数据的争议

首先使用 context 包主要是为了解决 goroutine 的通知退出，传递数据只是一个额外功能。而使用这个功能会存在以下问题

* 传递的都是 interface{} 类型的值，编译器不能进行严格的类型校验
* 从 interface{} 到具体类型需要使用类型断言和接口查询，这会带来一定的运行时开销和性能损失
* 值在传递过程中有可能被后续的服务覆盖，且不容易被发现（使用包私有类型可以解决`type userKey struct{}`）
* 传递的信息不简明，比较晦涩。不能通过代码或文档一眼看到传递的是什么，不利于后续维护。

最佳实践：使用 context 传递的信息不能影响正常的业务流程，且程序不要期待在 context 中获取到一些必须的参数等



### 并发模型

Go 语言主要借鉴 CSP 模型设计并发模型。CSP的基本思想是：将并发系统抽象为 Channel 和 Process 两部分，Channel 用来传递消息，Porcess 用于执行。Channel 和 Process 之间相互独立，没有从属关系，消息的发送和接受有严格的时序限制。

#### 并发和调度

Go 在语言层面引入 goroutine，有以下好处：

* goroutine 可以在用户空间调度，避免了内核态和用户态的切换导致的成本
* goroutine 是语言原生支持的，提供了非常简洁的语法，屏蔽了大部分复杂底层的实现
* goroutine 更小的栈空间允许用户创建成千上万的实例

在 Go 语言的并发模型中，Go 的并发模型抽象出三个实体：M、P、G

##### G（goroutine）

G 是 Go 运行时对 goroutine 的抽象，G 中存放并发执行体的代码入口地址、上下文、运行时环境（P 和 M）、运行栈等相关信息。

G 的新建、休眠、恢复、停止都受到 Go 运行时的管理。Go 运行时的监控线程会监控 G 的调度，所以 G 不会长久地阻塞系统线程。G 新建或恢复时会被添加到运行队列中，等到 M 取出并运行。

##### M（machine）

M 表示系统内核线程，是操作系统层面调度和执行的实体。M 仅负责执行，M 不停地被唤醒或创建，然后执行。M 在启动时会进入运行时的管理代码，这段代码毁负责获取 G 和 P 资源，然后执行调度。

另外 Go 语言还会单独创建一个监控线程，负责对程序的内存、调度等信息进行监控和控制。

##### P（processor）

P 表示 M 运行 G 所需要的资源，是一种对资源的抽象和管理。主要是为了降低 M 管理调度 G 的复杂性而增加的一个间接的控制层数据结构。把 P 看做资源而不是处理器，其中 P 种持有 G 的队列，P 可以隔离调度，解除 P 和 M 的绑定就解除了 M 对一串 G 的调度。

M 和 P 一起构成一个运行时环境，每个 P 都有一个本地的可调度 G 队列，队列里的 G 会被 M 依次调度执行，如果本地队列空了，则会去全局队列偷取一部分 G，如果全局队列也是空的，则会去其他的 P 中偷取一部分 G。这就是 Work Stealing 算法的基本原理。

##### 总结

G 只是保存并发执行体的元数据，其中包含入口地址、堆栈、上下文等。而 M 仅负责执行，在启动时进入运行时管理代码，这部分代码代码只有在拿到一个 P 之后才可以执行调度。P 的数量默认为 CPU 核心数量，也可通过 `runtime.GOMAXPROCS` 函数设置或查询。

#### g0 和 m0

m0 是启动程序后的主线程，这个 M 对应的信息会存放在全局变量 m0 中，其中 m0 负责执行初始化和启动第一个 G （也就是 g0）。

#### 什么时候创建 M、P、G

在程序启动过程中会初始化空闲的 P 列表，同时 主线程 m0 创建第一个 G （g0）。后续在有 `go` 并发调用的地方都有可能创建 G （G 可以被复用，会在 P 的空闲列表里面寻找已结束的 goroutine）。每个并发调用都会初始化一个新的 G 任务，然后唤醒 M 执行任务。
