类型系统
------------

类型系统能够让编译器在编译阶段发现大部分程序错误。



### 类型简介

类型可分为命名类型和未命名类型（又称类型字面量，复合类型）。命名类型是可以通过标识符来表示的类型，包括**Go预声明类型（简单类型）和用户自定义的类型**（使用type声明）。而未命名类型则是由预声明类型、关键字和操作符组合而成，在Go中的**复合类型**（数组、切片、字典、通道、指针、函数字面量、结构和接口）都属于未命名类型（与类型字面量，复合类型等价）。

```go
// Person 是一个命名类型
type Person struct {
	Name string
	Age  int
}

// logger 是一个使用未命名类型声明的变量
var logger struct {
	Level  int
	Format string
}

func main() {
	fmt.Printf("person => %T\n", Person{})
	fmt.Printf("logger => %T\n", logger)

	// Output:
	// 	person => main.Person
	//	logger => struct { Level int; Format string }
}
```

总结以上内容就是：

* 命名类型包含：Go语言中的简单类型（布尔、整形、浮点、复数、字符、字符串、接口）、用户自定义类型（通过`type`声明）
* 未命名类型包含：Go语言中的复合类型（数组、切片、字典、通道、指针、函数字面量、结构体、接口）

**注意以上两者都包含“接口”类型，接口也是可以匿名使用的，如下所示：**

```go
var printer interface {
    Print(string) error
}
```



#### 底层类型

在Go中所有的类型都有一个底层类型（`Underlying type`），底层类型的规则如下：

* 预声明类型（简单类型）和复合类型（类型字面量）的底层类型就是他们自身（*注意不要和命名类型混在一起*）
* 自定义类型的底层类型是逐层向下查找，直到找到预声明类型或复合类型为止

```go
// 底层类型为 string
type String string
// 底层类型为 string
type UTF8String String

// 底层类型为 []string
type StringList []string
// 底层类型为 []UTF8String
type UTF8StringList []UTF8String

// 注意：StringList 和 UTF8StringList 的底层类型并不相同
```

底层类型在类型赋值和类型强制转换时会使用



#### 相同类型与类型赋值

在Go中两个类型是否相同遵循以下规则：

* 具有相同的**命名类型**（类型的名字一致）
* 具有相同结构且内部元素的类型相同的**未命名类型**
* 通过**类型别名**声明的类型（`type T1 = T2`于Go 1.9引入）
* **命名类型和未命名类型永远不相同**

变量之间在满足一定条件情况下可以**直接赋值**：

* 两个变量的**类型相同**
* 两个变量**具有相同的底层类型且至少有一个为未命名类型**
* 其中一个变量实现了**另一个接口变量**的的所有方法
* 都是通道类型且具有相同的元素类型
* 其中一个值是nil，且另一个则是指针、函数、切片、字典、通道、接口类型的变量（`var a *int; a = nil`）
* 字面值可以被赋值给相对应类型（`var a int; a = 111`）

```go
type String string
type UTF8String string

var a String
var b UTF8String

a = b // 这里不能直接赋值，应该两个都是命名类型


type StringList []string

var al []string
var bl StringList

bl = al // 这里可以直接赋值，因为 al 是未命名类型
```



#### 类型强制转换

一个变量可以通过强制类型转换赋值给另一个**底层类型相同**的变量，或者是满足一些特殊情况（比如`string`和`[]byte`、`[]rune`之间的转换）。



### 类型方法

为类型增加方法是Go语言实现面对对象编程的基础。



#### 自定义类型

我们可以将自定义类型、预声明类型（int、float64、bool等）、未命名类型（[]T、struct{}）重新定义为一个新的**命名类型**。其中`struct`类型是Go语言自定义类型的普遍形式，这是Go语言类型扩展的基石，也是Go语言面对对象承载的基础。

```go
// 使用 type 定义一个命名类型
type TypeName struct{
    Field1 Type1
    Field2 Type2
}

// s 是一个未命名类型的变量
var s = struct{}{}
```

结构的字段可以是任何类型（包括基本类型、接口类型、指针类型、函数类型等），在结构中还支持内嵌自身的指针（这也是实现树形和链表等复杂数据结构的基础）。

在定义struct中，如果字段只给出类型而没有给出字段名，那么我们称这样的字段为“匿名字段”。匿名字段必须是命名类型或命名类型的指针，匿名字段的字段名默认就是类型名。



#### 方法

Go语言的类型方法是一种对**类型行为**的封装，我们可以将类型方法看做一个第一个参数为类型实例对象或指针的特殊函数。类型方法有以下特点：

* 可以为命名类型增加方法（除了接口），非命名类型不能自定义方法
* 为类型增加的方法必须**要和类型定义在同一个包中**
* 方法的可见性和变量一样，大写开头的方法可以在包的外部访问，而小写开头的方法只能在包内部使用
* 使用`type`定义的自定义类型是一个新类型，新类型不能调用原有类型的方法，但是底层类型支持的运算可以被新类型继承

```go
type Map map[string]string

func (m Map) Each() {
    for k, v := range m {
        // do something for k, v
    }
}
```



### 方法调用

方法调用的一般形式是`instance.MethodName(ParamList)`。除此之外，我们还可以通过方法值（`method value`）或者方法表达式（`method expression`）来调用方法。



#### 方法值

我们可以通过将实例（v）对应的类型（T）上的方法（M）赋值给一个变量（f），并通过`f(ParamList)`的方式调用。`v.M`被称为方法值（Method Value）。方法值其实就是一个函数类型的变量，可以向普通的函数一样使用。

方法值在底层的实现上就是一个带有闭包的函数变量，与其他带有闭包的匿名函数类似，接收器（receiver）被隐式绑定到方法值（Method Value）的闭包环境内。

```go
type Person struct {
	Name string
}

func (p *Person) SetName(name string) {
	p.Name = name
	fmt.Printf("&person = %p\n", p)
}

func main() {
	p := Person{Name: "foo"}
	p.SetName("bar")

	m := p.SetName
	m("baz")

	// output:
	//	&person = 0xc000040230
	//	&person = 0xc000040230
}
```



#### 方法表达式

方法表达式相当于提供一种语法将类型方法调用显式转换为函数调用，必须显式地传递接收者。

```go
type Person struct {
	Name string
}

func (p Person) GetName() string {
	return p.Name
}

func (p *Person) SetName(name string) {
	p.Name = name
	fmt.Printf("&person = %p\n", p)
}

func main() {
    // func(p Person) string
	g := Person.GetName
    // func(p *Person, name string)
	s := (*Person).SetName
	//s := Person.SetName
	// invalid method expression Person.SetName (needs pointer receiver: (*Person).SetName)

	p := Person{Name: "foo"}
	fmt.Printf("method expression: g() = %s\n", g(p))

	s(&p, "bar")
	fmt.Printf("method expression: g() = %s\n", g(p))

	// Output:
	// 	method expression: g() = foo
	//	&person = 0xc000040230
	//	method expression: g() = bar	
}
```

表达式`Person.GetName`和`(*Person).SetName`被称为方法表达式（Method Expression），这些方法得首个参数是接收器的实例或指针。需要注意这里接收器的类型需要与方法表达式的类型需要相匹配，否则编译器会报错（编译器不会在方法表达式里做自动转换）。



#### 方法集

无论接收者是什么类型，方法和函数的实参传递都是值拷贝。如果接收者是值类型，则传递的就是值的副本；如果接收者是指针类型，则传递的是指针的副本。

针对Golang中的方法集有以下结论：

```text
Values          Methods Receivers
-----------------------------------------------
T               (t T)
*T              (t T) and (t *T)

Methods Receivers    Values
-----------------------------------------------
(t T)                 T and *T
(t *T)                *T
```

>1. If you have a `*T` you can call methods that have a receiver type of `*T` as well as methods that have a receiver type of `T` (the passage you quoted, [Method Sets](https://golang.org/ref/spec#Method_sets)).
>2. If you have a `T` and it is [addressable](https://golang.org/ref/spec#Address_operators) you can call methods that have a receiver type of `*T` as well as methods that have a receiver type of `T`, because the method call `t.Meth()` will be equivalent to `(&t).Meth()` ([Calls](https://golang.org/ref/spec#Calls)).
>3. If you have a `T` and it isn't addressable (for instance, the result of a function call, or the result of indexing into a map), Go can't get a pointer to it, so you can only call methods that have a receiver type of `T`, not `*T`.
>4. If you have an interface `I`, and some or all of the methods in `I`'s method set are provided by methods with a receiver of `*T` (with the remainder being provided by methods with a receiver of `T`), then `*T` satisfies the interface `I`, but `T` doesn't. That is because `*T`'s method set includes `T`'s, but not the other way around (back to the first point again).
>
>In short, you can mix and match methods with value receivers and methods with pointer receivers, and use them with variables containing values and pointers, without worrying about which is which. Both will work, and the syntax is the same. However, if methods with pointer receivers are needed to satisfy an interface, then only a pointer will be assignable to the interface — a value won't be valid.



#### 值调用和表达式调用的方法集

1、通过类型字面量显式地进行值调用和表达式调用，在这种情况下编译器不会做自动转换，会进行严格的方法集检查。

2、通过类型变量进行值调用和表达式调用，在使用值调用（method value）的情况下编译器才会进行自动转换，使用表达式调用（method expression）方法时编译器不会进行转换而是进行严格的方法集检查。



### 组合和方法集

结构struct为Go提供了强大的类型扩展，第一，struct可以嵌入任意其他类型的字段；第二，struct可以嵌套自身的指针类型的字段。Go语言的结构体和C语言的struct一样，内存分配按照字段顺序依次开辟连续的存储空间，没有插入额外的东西（内存对齐除外）。这种设计的优点是数据和逻辑彻底分离，对象内存区只存放数据，干净简单；类型的方法也是显式带上接收者，没有像C++那样使用隐式的this指针。在Go中数据就是数据，逻辑就是逻辑，二则是正交的，底层实现上没有相关性，在语言使用层又为开发者提供了统一的数据和逻辑抽象视图，这种外部统一，内部隔离的面对对象设计是Go语言优秀设计的体现。



#### 组合

使用type定义的新类型不会继承原有类型的方法（但是可以继承底层类型支持的操作），内嵌字段与结构体之间是"has-a"的关系，而不是“is-a”的关系，这就是Go中的组合。没有父子的概念，仅仅是整体和局部的概念。

在访问内嵌字段时也是使用点操作符`.`，struct的字段可以嵌套很多层，只要内嵌的字段是唯一即可，不需要使用全路径进行访问。

```go
type A struct {
    X int
}

type B struct {
    A
    Y int
}

type C struct {
    B
    Z int
}

var c C
c.X = 1
c.Y = 2
c.Z = 3
```

在struct的多层嵌套中，不同嵌套层次也可以有相同的字段，此时最好使用完全路径进行访问和初始化。如果没使用全路径的话，**编译器会从外向内逐层查找的方式**。



#### 内嵌字段的方法调用

内嵌字段的方法调用也使用点操作符，与内嵌字段的查找方法一样，如果不同的内嵌层次有相同的方法也是从外向内逐层查找的。**同名方法中外层的方法能够覆盖内层的方法**。



#### 组合的方法集

组合结构的方法集有如下规则：

* 若类型S包含匿名字段T，则S的方法集包含T的方法集
* 若类型S包含匿名字段\*T，则S的方法集包含T和\*T的方法集
* 不管类型S包含匿名字段\*T还是T，\*S的方法集总是包含T和\*T的方法集

**注意：编译器的自动转换仅适用于直接通过类型实例调用方法，类型实例传递给接口时，编译器不会进行自动转换而是会进行严格的方法集校验。**

在Go中函数都是以值传递的方式调用的。如果传递给接口参数的是值类型，但调用方法的接收者是指针类型，则程序在运行时虽然能将接收者转换为指针，但是此时**指针是指向原本变量的副本的，并不是我们期望的原变量的指针**。所以语言设计者为了杜绝这种非期望的行为，在编译时做了严格的方法集的检查。

如果传递给接口的变量是指针类型，而接口的实现是值类型的方法，程序会自动将指针转换为值类型，这种转换不会带来副作用，符合调用者的预期，所以这种转换是被允许的。



### 函数类型

使用`func FunctionName(int) {}`语法定义的函数被称为“具名函数（有名函数）”，而直接通过`func() {}`方法定义的函数被称为“匿名函数”。函数类型也分为两种，一种是函数字面量类型（未命名类型），一种是命令函数类型。

#### 函数字面量类型

函数字面量类型的语法表达式为`func(FuncParamList) ReturnList {}`，可以看出具名函数和匿名函数都属于函数字面量类型。具名函数的定义相当于初始化一个函数字面量类型然后将其赋值给一个函数名变量；而匿名函数的定义也是直接初始化一个函数字面量类型，只是没有将其绑定到一个具体的变量上。

#### 命名函数类型

通过`type TypeName FuncLiteral`可以将一个未命名函数字面量定义为一个命令函数类型。其中`TypeName`是新定义的命名函数类型，而`FuncLiteral`是函数字面量类型，同时其作为`TypeName `的底层类型。

#### 函数签名

所谓“函数签名”就是具名函数或匿名函数的函数字面量类型（`func(int) `）。所以具名函数和匿名函数的函数签名可以相同。

#### 函数声明

在Go语言中一般不需要使用到函数声明，只有在一些特殊场景下（比如调用汇编）才会使用到。

```go
// add.go
// 函数声明 = 函数签名+函数名，但是没有函数体
func Add(a int, b int) int

// add.s
TEXT ·Add, NOSPLIT, $24-0
	MOVQ a+24(SP), AX
	// ...
```



#### 为函数添加方法

函数字面量是一个未命名类型，所以我们不能为其添加方法，必须显式地使用`type`定义一个命名函数类型，我们才能为其添加方法

```go
// net/http/server.go

type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}

// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as HTTP handlers. If f is a function
// with the appropriate signature, HandlerFunc(f) is a
// Handler that calls f.
type HandlerFunc func(ResponseWriter, *Request)

// ServeHTTP calls f(w, r).
func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) { // ** and here **
	f(w, r)
}

// HandleFunc registers the handler function for the given pattern.
func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	if handler == nil {
		panic("http: nil handler")
	}
	mux.Handle(pattern, HandlerFunc(handler)) // ** here **
}
```

通过以上对函数类型的使用，我们可以看到函数类型有如下意义：

* 函数也是一种类型，并且可以在函数字面量的基础上定义一种命名函数类型（`type TypeName FunctionLiteral`）。
* 具名函数和你们函数的签名与命名函数类型的底层类型相同，它们之间可以进行类型转换
* 可以为命名函数类型添加方法，可以方便地为一个函数增加“拦截”或“过滤”等额外功能（相当于提供一种装饰器模式）
* 为命名函数类型添加方法，打通了接口之间的关系，在使用接口的地方可以传递函数类型的变量。这为函数到接口的转换开启了大门。
