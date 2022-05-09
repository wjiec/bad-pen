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

这里还有一个特殊的实例

```go
type Any interface{}

func main() {
	var a interface{} = nil
	var b interface{} = a

	fmt.Printf("a == nil => %v\n", a == nil)
	fmt.Printf("b == nil => %v\n", b == nil)

	var c Any = nil
	var d interface{} = c

	fmt.Printf("c == nil => %v\n", c == nil)
	fmt.Printf("d == nil => %v\n", d == nil)
}
```



### 接口内部实现

接口是Go语言类型系统的灵魂，也是Go语言实现多态和反射的基础。

#### 接口的数据结构

接口变量必须初始化才有意义，没有初始化的接口变量的默认值是`nil`。把具体类型的实例传递给接口称为接口的实例化。在接口的实例化过程中，编译器通过特定的数据结构描述整个过程。

```go
//
// src/runtime/runtime2.go
//

type iface struct {
	tab  *itab
	data unsafe.Pointer
}

// layout of Itab known to compilers
// allocated in non-garbage-collected memory
// Needs to be in sync with
// ../cmd/compile/internal/reflectdata/reflect.go:/^func.WriteTabs.
type itab struct {
	inter *interfacetype
	_type *_type
	hash  uint32 // copy of _type.hash. Used for type switches.
	_     [4]byte
	fun   [1]uintptr // variable sized. fun[0]==0 means _type does not implement inter.
}


//
// src/runtime/type.go
//

type interfacetype struct {
	typ     _type
	pkgpath name
	mhdr    []imethod
}

type imethod struct {
	name nameOff
	ityp typeOff
}

// Needs to be in sync with ../cmd/link/internal/ld/decodesym.go:/^func.commonsize,
// ../cmd/compile/internal/reflectdata/reflect.go:/^func.dcommontype and
// ../reflect/type.go:/^type.rtype.
// ../internal/reflectlite/type.go:/^type.rtype.
type _type struct {
	size       uintptr
	ptrdata    uintptr // size of memory prefix holding all pointers
	hash       uint32
	tflag      tflag
	align      uint8
	fieldAlign uint8
	kind       uint8
	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	equal func(unsafe.Pointer, unsafe.Pointer) bool
	// gcdata stores the GC type data for the garbage collector.
	// If the KindGCProg bit is set in kind, gcdata is a GC program.
	// Otherwise it is a ptrmask bitmap. See mbitmap.go for details.
	gcdata    *byte
	str       nameOff
	ptrToThis typeOff
}

type nameOff int32
type typeOff int32
```

非空接口的底层数据结构是`iface`，代码位于`src/runtime/runtime2.go`

```go
//
// src/runtime/runtime2.go
//

type iface struct {
	tab  *itab // 存放类型及方法指针信息
	data unsafe.Pointer // 实例的副本的指针
}
```

非空接口初始化的过程就是初始化一个iface类型的结构。`iface`结构中只包含两个指针类型字段

* `itab`：用来存放接口**自身类型**（1）和**绑定的实例类型**（2）以及**实例相关的函数指针**（3）
* `data`：**指向接口绑定的实例的副本**，接口的初始化也是一种值拷贝（如果实例是一个指针类型，那么`data`就是指向“指针实例副本”的指针）。

在`itab`中是接口内部实现的核心和基础

```go
//
// src/runtime/runtime2.go
//

// layout of Itab known to compilers
// allocated in non-garbage-collected memory
// Needs to be in sync with
// ../cmd/compile/internal/reflectdata/reflect.go:/^func.WriteTabs.
type itab struct {
	inter *interfacetype // 接口自身的静态类型
	_type *_type // 接口绑定的具体实例的类型（动态类型）
    // 存放具体类型的哈希值
	hash  uint32 // copy of _type.hash. Used for type switches.
	_     [4]byte
	fun   [1]uintptr // variable sized. fun[0]==0 means _type does not implement inter.
}
```

在`itab`中有5个字段：

* `inner`：**指向接口自身元信息的指针**（在RO段）
* `_type`：是**指向接口绑定的具体类型元数据的指针**，`iface`里的`data`指针指向的是该类型的值。
* `hash`：是**接口绑定的具体类型的哈希值**，这个值是从`_type.hash`字段拷贝出来，这里冗余主要是为了**方便接口断言和接口查询时快速访问**。
* `fun`：是一个函数指针数组，可以理解为C++对象模型里面的虚拟函数指针。注意这个指针数组的大小是可变的，由编译器负责填充，运行时使用底层指针进行访问（不受数组越界的约束），**数组里的指针指向的是具体类型的方法**。

`itab`这个数据结构是非空接口实现动态调用的基础，`itab`里的信息被编译器和链接器保存在可执行文件的RO段中。`itab`存放在静态分配的存储空间中，不受到GC的限制，其内存也不会被回收。

由于Go语言是一种强类型语言，编译器在编译时会做严格的类型校验。所以Go需要为每种类型维护相关的元信息（在运行时和反射都会用到）。而**Go语言的类型元信息的通用结构就是`_type`，其他类型都是以`_type`为内嵌字段封装而成的结构体**。

```go
//
// src/runtime/type.go
//

// Needs to be in sync with ../cmd/link/internal/ld/decodesym.go:/^func.commonsize,
// ../cmd/compile/internal/reflectdata/reflect.go:/^func.dcommontype and
// ../reflect/type.go:/^type.rtype.
// ../internal/reflectlite/type.go:/^type.rtype.
type _type struct {
	size       uintptr // 类型的大小
	ptrdata    uintptr // size of memory prefix holding all pointers
	hash       uint32 // 类型的哈希值
	tflag      tflag // 类型的特征标记
	align      uint8 // _type作为整体保存时的对齐字节数
	fieldAlign uint8 // 当前结构字段的对齐字节数
	kind       uint8 // 基础类型的枚举值，与 reflect.Kind 的值相同，决定了如何解析该类型
	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	equal func(unsafe.Pointer, unsafe.Pointer) bool
	// gcdata stores the GC type data for the garbage collector.
	// If the KindGCProg bit is set in kind, gcdata is a GC program.
	// Otherwise it is a ptrmask bitmap. See mbitmap.go for details.
	gcdata    *byte // GC的相关信息
	str       nameOff // 用来表示该类型的名称字符串在二进制文件中的偏移值。由链接器负责填充
	ptrToThis typeOff // 用来表示类型元信息的指针在编译后二进制文件中的偏移值。由链接器负责填充
}
```

`_type`包含所有类型的共同元数据，编译器和运行时可以根据该元信息解析具体类型（类型名、类型的哈希值等）的基本信息。

在`_type`中的`str nameOff`和`ptrToThis typeOff`最终都是由链接器负责确定和填充的。运行时提供了两个转换查找函数

```go
//
// src/runtime/type.go
//

// 获取 _type 的名字
func resolveNameOff(ptrInModule unsafe.Pointer, off nameOff) name {}

// 获取 _type 的副本
func resolveTypeOff(ptrInModule unsafe.Pointer, off typeOff) *_type {}
```

**Go语言的类型元信息是由编译器负责构建，并以表的形式存放在编译后的对象文件中。链接器在链接时进行段合并、符号重定位（填充某些值）。然后这些类型信息在接口的动态调用和反射中被引用。**

以下为接口类型在`_type`类型之上封装的额外元数据类型

```go
//
// src/runtime/type.go
//

type interfacetype struct {
	typ     _type // 类型的通用元数据
	pkgpath name // 包的名称
	mhdr    []imethod // 接口的方法签名列表
}

type imethod struct {
	name nameOff // 方法的名字在二进制文件中的偏移
	ityp typeOff // 方法类型元数据在二进制文件中的偏移
}
```

#### 接口调用过程分析

我们首先准备以下代码并将其反编译来分析接口实例化和动态调用过程

```go
package main

type Calculator interface {
	Add(a, b int) int
	Sub(a, b int) int
}

type Simpler struct {
	ID int64
}

//go:noinline
func (s Simpler) Add(a, b int) int {
	return a + b
}

//go:noinline
func (s Simpler) Sub(a, b int) int {
	return a - b
}

func main() {
	var c Calculator = Simpler{ID: 1234}
	c.Add(77, 88)
}

// go tool compile -N -l -S main.go > main.S
```

接下来我们首先看 main 函数的汇编代码

```asm
// func main()
"".main STEXT size=253 args=0x0 locals=0x60 funcid=0x0
    0x0000 00000 (main.go:23)    TEXT    "".main(SB), ABIInternal, $96-0
    // 检查是否需要进行栈扩展
    0x0000 00000 (main.go:23)    MOVQ    (TLS), CX
    0x0009 00009 (main.go:23)    CMPQ    SP, 16(CX)
    0x000d 00013 (main.go:23)    JLS    243

    // 为 main 函数准备栈空间
    0x0013 00019 (main.go:23)    SUBQ    $96, SP
    0x0017 00023 (main.go:23)    MOVQ    BP, 88(SP)
    0x001c 00028 (main.go:23)    LEAQ    88(SP), BP

    // 创建对象 Simpler{A: 1234, B: 5678}
    0x0021 00033 (main.go:24)    MOVQ    $0, ""..autotmp_1+56(SP)    // 给 A 字段赋零值
    0x002a 00042 (main.go:24)    MOVL    $0, ""..autotmp_1+64(SP)    // 给 B 字段赋零值
    0x0032 00050 (main.go:24)    MOVQ    $1234, ""..autotmp_1+56(SP) // 初始化字段 A 的值为 1234
    0x003b 00059 (main.go:24)    MOVL    $5678, ""..autotmp_1+64(SP) // 初始化字段 B 的值为 5678

    //
    // src/runtime/runtime2.go
    //
    // type iface struct {
    //     tab  *itab // 存放类型及方法指针信息
    //     data unsafe.Pointer // 实例的副本的指针
    // }
    //

    // 因为 Simpler 的接收者为 <值类型> 所以这里会拷贝一份 Simpler
    0x0043 00067 (main.go:24)    MOVQ    ""..autotmp_1+56(SP), AX    // 获取 A 字段的值并赋值给 AX = 1234
    0x0048 00072 (main.go:24)    MOVQ    AX, ""..autotmp_2+40(SP)    // 初始化字段 c.A 的值为 AX = 1234
    0x004d 00077 (main.go:24)    MOVL    $5678, ""..autotmp_2+48(SP) // 初始化字段 c.B 的值为 5678

    // 初始化接口变量 var c Calculator = ...
    //
    // itab: 存放接口 自身类型 和 绑定的实例类型 以及 实例相关的函数指针
    0x0055 00085 (main.go:24)    LEAQ    go.itab."".Simpler,"".Calculator(SB), AX   // 获取 Simpler 类型对应 Calculator 接口的 itab 地址
    0x005c 00092 (main.go:24)    MOVQ    AX, "".c+72(SP)                            // 为 c.tab 字段赋值
    0x0061 00097 (main.go:24)    LEAQ    ""..autotmp_2+40(SP), AX                   // 获取拷贝的 Simpler 对象的地址
    0x0066 00102 (main.go:24)    MOVQ    AX, "".c+80(SP)                            // 为 c.data 赋值, 注意这里是指针, 所以取的是地址赋值

    // 销毁首次创建的 Simpler 对象
    0x006b 00107 (main.go:25)    MOVQ    $0, ""..autotmp_1+56(SP)
    0x0074 00116 (main.go:25)    MOVL    $0, ""..autotmp_1+64(SP)

    // 这里不知道为啥要检查一下接口变量 c 中的 itab 位置是否相同?
    0x007c 00124 (main.go:25)    MOVQ    "".c+72(SP), AX
    0x0081 00129 (main.go:25)    MOVQ    "".c+80(SP), CX
    0x0086 00134 (main.go:25)    LEAQ    go.itab."".Simpler,"".Calculator(SB), DX
    0x008d 00141 (main.go:25)    CMPQ    DX, AX
    0x0090 00144 (main.go:25)    JEQ    148         // 如果 AX == DX, 则继续执行
    0x0092 00146 (main.go:25)    JMP    209         // 否则抛出异常

    0x0094 00148 (main.go:25)    MOVL    8(CX), AX
    0x0097 00151 (main.go:25)    MOVQ    (CX), CX
    0x009a 00154 (main.go:25)    MOVQ    CX, ""..autotmp_1+56(SP)
    0x009f 00159 (main.go:25)    MOVL    AX, ""..autotmp_1+64(SP)
    0x00a3 00163 (main.go:25)    MOVQ    ""..autotmp_1+56(SP), CX
    0x00a8 00168 (main.go:25)    MOVQ    CX, (SP)
    0x00ac 00172 (main.go:25)    MOVL    AX, 8(SP)
    0x00b0 00176 (main.go:25)    MOVQ    $77, 16(SP)
    0x00b9 00185 (main.go:25)    MOVQ    $88, 24(SP)
    0x00c2 00194 (main.go:25)    CALL    "".Simpler.Add(SB)
    0x00c7 00199 (main.go:26)    MOVQ    88(SP), BP
    0x00cc 00204 (main.go:26)    ADDQ    $96, SP
    0x00d0 00208 (main.go:26)    RET

    // 接口变量 c.tab 的指针与 实际的类型指针位置不相同触发 panic
    0x00d1 00209 (main.go:25)    MOVQ    AX, (SP)                   // 第一个参数对应 c.tab
    0x00d5 00213 (main.go:25)    LEAQ    type."".Simpler(SB), AX
    0x00dc 00220 (main.go:25)    MOVQ    AX, 8(SP)                  // 第二个参数对应 Simpler._type
    0x00e1 00225 (main.go:25)    LEAQ    type."".Calculator(SB), AX
    0x00e8 00232 (main.go:25)    MOVQ    AX, 16(SP)                 // 第三个参数对应 Calculator._type
    0x00ed 00237 (main.go:25)    CALL    runtime.panicdottypeI(SB)  // func panicdottypeI(have, want, iface *byte)
    0x00f2 00242 (main.go:25)    XCHGL    AX, AX
    0x00f3 00243 (main.go:25)    NOP

    // 执行栈扩展方法, 扩展完成后返回函数入口再次检查
    0x00f3 00243 (main.go:23)    CALL    runtime.morestack_noctxt(SB)
    0x00f8 00248 (main.go:23)    JMP    0
```

