defer 延迟调用
---------------------

 defer 是 Go 语言中的关键字，也是 Go 语言的重要特性之一。defer 之后必须紧跟一个函数调用或者方法调用，且不能用括号括起来。在很多时候，defer 后的函数都是以匿名函数或闭包的形式呈现。

defer 将其后的函数推迟到其所在函数返回之前执行，不管 defer 语句后的执行路径如何，最终都将被执行。在 Go 语言中，defer 一般被用于资源的释放及异常 panic 的捕获处理。



### 使用 defer 的优势



#### 释放资源

作为 Go 语言的特性之一，defer 给 Go 代码的编写方式带来了很大的变化：

```go
func CopyFile(dst, src string) (int64, error) {
    sfp, err := os.Open(src)
    if err != nil {
        return 0, err
    }
    defer func() { _ = src.Close() }()
    
    dfp, err := os.Create(dst)
    if err != nil {
        return 0, err
    }
    defer func() { _ = dfp.Close() }()

    return io.Copy(dfp, sfp)
}
```

defer 是一种优雅的关闭资源的方式，能减少大量冗余的代码并避免由于忘记释放资源而产生的错误。



#### 异常捕获

程序在运行时可能在任意的地方发生 panic 异常，同时这些错误会导致程序异常退出。在很多时候，我们希望能够捕获这样的错误，同时希望程序能够继续正常执行。而 defer 为异常补货提供了很好的时机，一般和 recover 函数结合在一起使用。



### defer 特性

defer 后的函数不会立即执行，而是推迟到函数结束后再执行。这一特性一般用于资源的释放，除了可以用在资源释放和异常捕获上，有时也可以用于函数的中间件。



#### 参数预计算

defer 在使用时，延迟调用函数的参数将立即求值，传递到 defer 函数中的参数将预先被固定，而不会等到函数执行完成后再传递到 defer 函数中。

```go
func main() {
    i := 100
    
    defer func(n int) {
        fmt.Printf("defer n = %d\n", n)
    }(i + 1)
    
    i = 200
    fmt.Printf("main i = %d\n", i)
    // main i = 200
    // defer n = 101
}
```



#### defer 多次执行与 LIFO 执行顺序

在函数体内部出现的多个 defer 函数将会按照后入先出（Last-In First-Out）的顺序执行。

```go
func main() {
    defer func() { fmt.Println("hello") }()
    defer func() { fmt.Println("world") }()
    defer func() { fmt.Println("foo") }()
    defer func() { fmt.Println("bar") }()
    // bar
    // foo
    // world
    // hello
}
```



#### 返回值陷阱

当 defer 与返回值相结合时，需要注意返回语句的语义问题：

```go
var g = 100

func f1() (r int) {
    defer func() { g = 200 }()

    return g
}

func f2() (r int) {
    defer func() { r = 300 }()

    return g
}

func main() {
    fmt.Printf("f1() = %d\n", f1())
    fmt.Printf("f2() = %d\n", f2())
    // f1() = 100
    // f2() = 300
}
```

以上问题的原因在于，return 其实并不是一个原子操作，其包含了以下几个步骤：

* 将返回值保存到栈上
* 执行 defer 语句
* 函数执行 RET 返回



### defer 底层原理

