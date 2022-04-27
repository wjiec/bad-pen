接口
------

在Go语言中，接口是一个编程规约，也是一组方法签名（函数字面量类型，不包括函数名）的集合。一个具体类型的方法集是接口方法集的超级，就代表类型实现了这个接口（不需要在语法上进行显式的声明）。**编译器会在编译时进行方法集的校验**。

最常使用的接口字面量类型是空接口（`interface{}`），由于空接口的方法集为空，所以任何类型都可以被认为实现了空接口。任意类型的实例都可以赋值或传递给空接口，包括未命名类型。



### 基本概念

接下来主要介绍

#### 接口声明

Go语言中接口分为接口字面量和接口命名类型，接口的声明使用`interface`关键字

```go
// 接口字面量
interface {
    Write([]byte) (int, error)
}

// 接口字面量的使用
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    _, _ = w.(interface {
        Write([]byte) (int, error)
    })
})

var r interface {
    Read(p []byte) (int, error)
}
r = os.Stdin

// 空接口也是一个接口字面量
interface{}

// 接口命名类型
type InterfaceName interface {
    FunctionSignature()
}

// 也可以嵌入另一个接口类型
type EmbeddedInterface interface {
    io.Reader
    io.Writer
}
```

**Go编译器在做接口匹配判断时时严格校验方法名称和方法签名（方法字面量，不包括方法名）的**。

#### 接口初始化

接口只有在初始化为具体的类型时才有意义。没有初始化的接口变量，其默认值是nil。接口绑定具体类型的实例的过程被称为接口初始化，有以下两种方式

1、**将实例赋值给接口变量**

如果某个具体类型的方法集是某个接口方法集的超集，那我们就可以将该类型的实例直接赋值给接口类型的变量，**此时编译器会进行静态的类型检查**。接口在被初始化之后，**调用接口的方法就相当于调用接口绑定的具体类型的方法**。

2、**将已初始化的接口赋值给另一个接口**

如果B接口的方法集是A接口方法集的子集，那么我们可以将已初始化的A接口类型变量a直接赋值给B接口类型变量b，此时编译器会在编译时进行方法集的静态检查。这种情况下**接口变量b的具体实例是接口变量a绑定的具体实例的副本**。

```go
type Apple interface {
	App()
}

type Banana interface {
	Apple

	Ban()
}

type Sample struct{}

func (Sample) App() {}
func (Sample) Ban() {}

func main() {
    // 这个例子不好，没想到啥好的解释方法
	s := Sample{}
	fmt.Printf("&s = %p\n\n", &s)

	var b Banana = s
	fmt.Printf("&s = %p\n", &s)
	fmt.Printf("&b = %p\n\n", &b)

	var a Apple = b
	fmt.Printf("&s = %p\n", &s)
	fmt.Printf("&b = %p\n", &b)
	fmt.Printf("&a = %p\n\n", &a)
}
```

#### 调用接口方法

接口方法调用的最终地址是在运行前决定的，**将具体类型变量赋值给接口之后，会使用具体类型的方法指针初始化接口变量**。当调用接口方法时，实际上是间接地调用实例的方法（**有一定的运行时开销**）。

#### 接口的动态类型和静态类型

接口的动态类型：接口**绑定的具体实例的类型**成为接口的动态类型（接口的动态类型随着其绑定的具体实例类型而变化）。

接口的静态类型：接口被定义时的类型被称为接口的静态类型，静态类型的本质特征就是接口的方法签名集合（方法集）。

**如果两个类型的方法签名（不带名字的方法字面量）集合相同（顺序可以不一样），他们之间就不需要强制类型转换就可以相互赋值。原因是Go编译器校验接口是否能赋值比较的是二者的方法集，而不是看具体接口类型名。**



### 接口运算

为了知道已经初始化的接口变量绑定的具体类型，以及这个具体实例是否还实现了其他接口。Go语言提供两种语法结构来支持这两种需求，分别是**类型断言**和**类型接口查询**。

#### 类型断言（Type Assertion）

类型断言的基本语法形式如下：

```go
i.(TypeName) // i 必须是接口变量，否则编译器会报错
```

接口断言的两层含义：

1、如果`TypeName`是一个具体类型名，则类型断言用于判断接口变量 `i` 绑定的实例类型是否就是具体类型 `TypeName`

2、如果`TypeName`是一个接口类型名，则类型断言用于判断接口变量 `i` 绑定的实例类型是否实现了`TypeName`接口

##### 接口断言的复制形式

```go
// 如果成功断言，则 v 是接口绑定类型实例的副本（断言类型）或是底层绑定具体类型实例的副本的接口变量（断言接口）
// 如果断言失败，则直接 panic
v := i.(TypeName)

// 如果成功断言，则 v 是接口绑定类型实例的副本（断言类型）或是底层绑定具体类型实例的副本的接口变量（断言接口）且 ok = true
// 如果断言失败，不会发生 panic，且 ok = false 同时 v 为 TypeName 类型的零值（断言类型则是具体类型的零值，断言接口则是 nil）
v, ok := i.(TypeName)
```

#### 类型查询（Type Switches）

类型查询和类型断言具有相同的语义，只是语法格式不同。同时类型查询使用`case`语句一次判断多个类型，而类型断言一次只能判断一个类型（可以通过if else来实现相同的效果）。类型查询的语法格式如下

```go
switch v := i.(type) { // v 可以被省略
case nil: // 如果 i 是未初始化的接口变量，则匹配这个子句
    // i == nil
case io.Reader:
    // typeof v == io.Reader
case *os.File:
    // typeof v == *os.File
case A, *B, C:
    // v == i
}
```

**同样这里的`i`也必须是接口类型，因为具体类型的实例的类型是静态的，在声明之后类型就不再变化，所以具体类型的变量不存在类型查询**。

类型查询的`case`语句后面可以跟非接口类型名，也可以跟接口类型名，**匹配是按照case子句的顺序进行的**。

1、如果`case`后面跟的是一个接口名，且接口变量`i`绑定的实例类型实现了该接口类型的方法，则成功匹配（`v`为该接口类型的变量）

2、如果`case`后面跟的是一个类型名，且接口变量`i`绑定的实例类型与该类型相同，则成功匹配（`v`为该具体类型的实例）

3、如果`case`后面跟多个使用`,`分割的类型，只要接口变量`i`与其中任一类型相同，则直接将变量`i`赋值给`v`（相当于`v := i`）

4、如果所有的`case`子句都不满足，则执行default语句，此时执行的任然是`v := i`

#### 接口的优点和使用形式

接口主要有以下优点

1、**解耦**：这是一种对复杂系统进行垂直和水平分割的常用手段，在层与层之间使用接口进行抽象和解耦是一种好的编程策略。同时由于Go中接口的非侵入式设计使层与层之间的代码更加干净，增加了接口使用的自由度。

2、**实现泛型**：在没有泛型的情况下，使用空接口作为函数参数或者返回值能够满足一部分的泛型需求（Go在1.18中已实现泛型支持）。

接口作为“一等公民”，可以用在任何使用变量的地方，主要的使用形式如下：

1、**作为结构内嵌字段**：表示该结构实现了这个接口

2、**作为函数或方法的形参、返回值**：用于实现动态绑定和面向接口编程，解耦不同层之间的实现，实现控制反转、依赖注入等

3、**作为其他接口定义的嵌入字段**：扩展接口的方法签名，实现接口之间的组合



### 空接口

没有任何方法的接口（`interface{}`，未命名类型）我们称之为空接口。空接口一般用来作为弥补泛型的一种手段（在Go1.18已增加泛型特性），同时空接口也是反射实现的基础。

#### 空接口和nil

空接口并不是完全的空，接口有类型和值两个概念（字段）。看如下例子：

```go
type Printer interface {
	Print()
	Println()
}

type Apple struct{}

func (a Apple) Print() {
	fmt.Println("Apple.Print")
}

func (a *Apple) Println() {
	fmt.Println("Apple.Println")
}

func main() {
	var apple *Apple = nil
	var printer Printer = apple

	fmt.Printf("apple = %p\n", apple)
	fmt.Printf("printer %p\n", printer)

	if printer != nil {
		printer.Println()

        // panic: runtime error: invalid memory address or nil pointer dereference
		//printer.Print()
	}
    
    // Output:
    //	apple = 0x0
	//	printer 0x0
	//	Apple.Println
}
```

注意这里我们打印接口`printer`的地址时，输出的结果是`0x0`（这表示**接口绑定的实例的地址**），而实际上接口中有2个字段，一个是实例类型，一个是指向绑定实例的指针。只有两个都为`nil`时，空接口才为`nil`。

```go
type Interface struct {
    Type TypeInfo
    Bind *Instance
}

var a interface{} // &Interface{Type: nil, Bind: nil}

var xxx *int = nil
var b interface{} = xxx // &Interface{Type: *int, Bind: nil}
```

