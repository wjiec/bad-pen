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

