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



### panic 和 recover

有两种情况可以引发panic，一种是程序主动调用panic函数，另一种是程序产生运行时错误时由运行时检测并抛出。发生panic之后，程序会直接从调用panic函数处立即返回，并逐层向上执行函数的defer语句（如果有的话），直到**被derfer中的recover函数**捕获或退回到main函数最后异常退出。

recover用来捕获panic，在通过recover捕获异常之后还可以再次调用panic抛出异常。同时注意**recover只有出现在derfer函数之中才能捕获panic**且不能嵌套。

```go
func main() {
	defer func() {
		fmt.Printf("first: %v\n", recover())
	}()

	defer func() {
		func() {
			fmt.Printf("second: %v\n", recover())
		}()
	}()

	defer fmt.Printf("third: %v\n", recover())

	defer recover()

	panic("panic message")
	// third: <nil>
	// second: <nil>
	// first: panic message
}
```

init函数引发的panic只能在init函数中捕获，无法在main中捕获，因为init函数会在main函数之前执行。**函数并不能捕获在其中启动的goroutine所抛出的panic**。

#### 使用场景

panic一般有以下两种情况会使用到：

1、程序遇到了无法正常执行下去的错误，主动调用panic结束程序

2、在调试程序时，通过主动调用panic快速退出程序，同时panic打印出的堆栈信息能够更快速的定位错误。



### 错误处理

Go语言内置错误接口类型为`error`，在Go中典型的错误处理方式是将error作为函数的最后一个返回值。

#### 处理错误和异常

错误和异常可以按如下方式进行区分：

* 广义的错误：发生了非预期的行为
* 狭义的错误：发生了已知且预期的行为
* 异常：发生了非预期的未知行为

因为Go是一门类型安全的语言，其运行时一般不会出现一些编译或运行时都无法捕获的异常。所以在Go中需要处理的错误可以分为两类：

* 运行时错误：由运行时捕获并隐式或显式抛出
* 程序逻辑错误：程序的执行结果不符合预期，但是不会引发异常

在实际的编程中，error和panic应该遵循以下规则：

* 程序发生的错误导致程序不能继续执行下去了，此时程序应该主动调用panic或由运行时抛出异常
* 程序虽然发生错误，但是是可预期且能恢复的，那就应该使用返回值的方式处理错误，或者在可能发生运行时错误的地方使用recover捕获panic。



### 底层实现

研究底层实现有两种方法，一种是看语言编译器源码，分析其对函数的各个特性的处理逻辑，另一种是反汇编，将可执行程序反汇编出来。

Go编译器产生的汇编代码是一种中间抽象态，并不是真实的机器码，而是和平台无关的一种中间态汇编描述。所以汇编代码中有些寄存器是真实的，有些是抽象的。抽象寄存器如下：

* `SB(Static Base Pointer)`：静态基址寄存器，表示全局符号的起始位置
* `FP(Frame Pointer)`：栈帧寄存器，该寄存器指向当前函数调用栈帧的栈底位置
* `SP(Stack Pointer)`：栈顶寄存器，一般在函数调用前由主调函数通过设置SP的值对栈空间进行分配和回收
* `PC(Program Counter)`：程序计数器，存放下一条指令的执行地址

#### 反汇编分析

我们可以通过以下命令来生成汇编代码（参考[Plan9][http://9p.io/sys/doc/asm.html]）

```bash
# -S 生成汇编代码
# -N 禁止优化
# -l 禁用内联
go tool compile -S -N -l name.go > name.S
```

我们可以从汇编的代码得知：

1、函数的调用者负责环境准备，包括为参数和返回值开辟栈空间

2、寄存器的保存和恢复也由调用方负责

3、函数调用后回收栈空间，恢复`BP`也由主调函数负责

函数的返回值实质上是在栈上开辟多个地址分别存放返回值。如果返回值是存放在堆上，则多了一个复制的动作。

#### 闭包的底层实现

闭包的实现是将函数指针和对应的参数包装成一个结构体的指针并返回。调用时直接调用函数指针所指向的位置。
