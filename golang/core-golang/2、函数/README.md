函数
------

几乎所有的高级语言都支持函数和类似函数的结构，函数之所以如此普遍和重要的原因如下：第一，现代计算机进程执行模型大部分是基于“堆栈”的，而编译器不需要对函数做过多的转换就能让其在栈上运行（只需要处理好参数和返回值的传递即可）；第二，函数对代码的抽象程度适中，就像胶水，很容易将编程语言的不同层级的抽象体“黏结起来”。



### 关于函数的一些特性

1、Go函数实参到形参的传递永远都是值拷贝（包括切片和字典）

2、可变参数在函数体内相当于切片，对切片的操作同样适合于对可变参数

3、形参为可变参数的函数和形参为切片的函数类型不相同



### defer

Go函数里提供了`defer`关键字，可以注册多个延迟调用，这些调用以先进后出（FILO）的顺序在函数返回前被执行。`defer`后面必须是函数或方法的调用，不能是语句，否则会报`expression in defer must be function call`。注意以下例子输出

```go
type Actor func(string) Actor

func Do(s string) Actor {
    fmt.Println(s)

    return Do
}

func TestDefer() {
    defer Do("A")("B")("C")("D")
}

func main() {
    // A B C D
    TestDefer()
}
```



`defer`语句必须先注册后才能执行，如果`defer`位于`return`之后，则`defer`会因为没有注册而不会执行。**当主动调用`os.Exit(status)`退出进程时，即使已经注册的`defer`也不会执行**。

`defer`的好处是可以在一定程度上避免资源泄露，特别是在有很多`return`语句且有很多资源需要关闭的场景中。**同时也要注意到`defer`会推迟资源的释放时间，可能导致大量内存或系统资源得不到释放。另外，`defer`相对于普通的函数调用需要间接的数据结构支持，相对于普通函数调用有一定的性能损耗**。





### 闭包

闭包是由**函数**及其所**引用的数据**组合而成而实体。闭包对闭包外的数据是直接引入，一旦编译器检测到闭包，就会将闭包所引用的外部变量分配到堆上（内存逃逸）。

**对象是带有行为的数据，而闭包是带有数据的行为**。

#### 闭包引用数据的规则

1、函数返回的闭包所引用的局部变量是不同的副本（形参其实也是一个局部变量，每次调用函数都会为局部变量分配内存）

2、闭包所引用的数据在多次调用时是共享的，即可以多次修改引用的数据

```go
type Action func(int) int

func Accumulate(base int) Action {
    return func(i int) int {
        fmt.Printf("&base = %p\n", &base)
        base += i
        return base
    }
}

func main() {
    add1 := Accumulate(1)
    add2 := Accumulate(1)

    fmt.Println("add1(10) =", add1(10))
    fmt.Println("add2(20) =", add2(20))

    fmt.Println("add1(30) =", add1(30))
    fmt.Println("add2(40) =", add2(40))
    // &base = 0xc000012088
    // add1(10) = 11
    //
    // &base = 0xc0000120a0
    // add2(20) = 21
    //
    // &base = 0xc000012088
    // add1(30) = 41
    //
    // &base = 0xc0000120a0
    // add2(40) = 61
}
```

3、在闭包中修改全局变量对所有闭包均可见

4、多个闭包所引用的局部变量是共享的

```go
type Action func(int) int

func Accumulate(base int) Action {
    return func(i int) int {
        fmt.Printf("&base = %p\n", &base)
        base += i
        return base
    }
}

func Combine(base int) (Action, Action) {
    return func(i int) int {
            base += i
            return base
        }, func(i int) int {
            base -= i
            return base
        }
}

func main() {
    c1, c2 := Combine(0)
    fmt.Printf("c1(111) = %d\n", c1(111))
    fmt.Printf("c2(222) = %d\n", c2(222))

    fmt.Printf("c1(111) = %d\n", c1(111))
    fmt.Printf("c2(222) = %d\n", c2(222))
    
    // c1(111) = 111
    // c2(222) = -111
    // c1(111) = 0
    // c2(222) = -222
}
```



