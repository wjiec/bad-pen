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
    0x0032 00050 (main.go:24)    MOVQ    $1234, ""..autotmp_1+56(SP) // 初始化字段 A 的值为 1234(int64)
    0x003b 00059 (main.go:24)    MOVL    $5678, ""..autotmp_1+64(SP) // 初始化字段 B 的值为 5678(int32)

    // 因为 Simpler 的接收者为 <值类型> 所以这里会拷贝一份 Simpler
    0x0043 00067 (main.go:24)    MOVQ    ""..autotmp_1+56(SP), AX    // 获取 A 字段的值并赋值给 AX = 1234(int64)
    0x0048 00072 (main.go:24)    MOVQ    AX, ""..autotmp_2+40(SP)    // 初始化字段 c.A 的值为 AX = 1234(int64)
    0x004d 00077 (main.go:24)    MOVL    $5678, ""..autotmp_2+48(SP) // 初始化字段 c.B 的值为 5678(int32)

    //
    // src/runtime/runtime2.go
    //
    // type iface struct {
    //     tab  *itab // 存放类型及方法指针信息
    //     data unsafe.Pointer // 实例的副本
    // }
    //
    // itab: 存放接口 自身类型 和 绑定的实例类型 以及 实例相关的函数指针
    //
    // 初始化接口变量 var c Calculator = ...
    0x0055 00085 (main.go:24)    LEAQ    go.itab."".Simpler,"".Calculator(SB), AX   // 获取 Simpler 类型对应 Calculator 接口的 itab 地址
    0x005c 00092 (main.go:24)    MOVQ    AX, "".c+72(SP)                            // 为 c.tab 字段赋值
    0x0061 00097 (main.go:24)    LEAQ    ""..autotmp_2+40(SP), AX                   // 获取拷贝的 Simpler 对象的地址
    0x0066 00102 (main.go:24)    MOVQ    AX, "".c+80(SP)                            // 为 c.data 赋值, 注意这里是指针, 所以取的是地址赋值

    // 清理首次创建的 Simpler 对象, 因为在栈上, 所以直接赋零值就行
    0x006b 00107 (main.go:25)    MOVQ    $0, ""..autotmp_1+56(SP)
    0x0074 00116 (main.go:25)    MOVL    $0, ""..autotmp_1+64(SP)

    // 这里不知道为啥要检查一下接口变量 c 中的 itab 指针是否相同?
    0x007c 00124 (main.go:25)    MOVQ    "".c+72(SP), AX        // AX = c.tab
    0x0081 00129 (main.go:25)    MOVQ    "".c+80(SP), CX        // CX = c.data
    0x0086 00134 (main.go:25)    LEAQ    go.itab."".Simpler,"".Calculator(SB), DX
    0x008d 00141 (main.go:25)    CMPQ    DX, AX
    0x0090 00144 (main.go:25)    JEQ    148         // 如果 AX == DX, 则继续执行
    0x0092 00146 (main.go:25)    JMP    209         // 否则抛出异常

    0x0094 00148 (main.go:25)    MOVL    8(CX), AX          // AX = c.data.(*Simpler).B (int32)
    0x0097 00151 (main.go:25)    MOVQ    (CX), CX           // CX = c.data.(*Simpler).A (int64)
    0x009a 00154 (main.go:25)    MOVQ    CX, ""..autotmp_1+56(SP)   // 复用初次创建的 Simpler 对象, simpler.A = CX
    0x009f 00159 (main.go:25)    MOVL    AX, ""..autotmp_1+64(SP)   // 复用初次创建的 Simpler 对象, simpler.B = AX
    0x00a3 00163 (main.go:25)    MOVQ    ""..autotmp_1+56(SP), CX   // 重新获取 simpler.A 的值并赋值给 CX
    0x00a8 00168 (main.go:25)    MOVQ    CX, (SP)           // Simpler.Add 的接收者对应的 s.A 字段
    0x00ac 00172 (main.go:25)    MOVL    AX, 8(SP)          // Simpler.Add 的接收者对应的 s.B 字段
    0x00b0 00176 (main.go:25)    MOVQ    $77, 16(SP)        // Simpler.Add 的第一个参数
    0x00b9 00185 (main.go:25)    MOVQ    $88, 24(SP)        // Simpler.Add 的第二个参数
    0x00c2 00194 (main.go:25)    CALL    "".Simpler.Add(SB) // 直接调用 Simpler.Add 方法

    // 回收 main 函数的栈空间并返回
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

    // 执行栈扩展方法, 扩展完成后返回函数入口再次检查
    0x00f3 00243 (main.go:25)    NOP
    0x00f3 00243 (main.go:23)    CALL    runtime.morestack_noctxt(SB)
    0x00f8 00248 (main.go:23)    JMP    0
```

**注意：以上直接调用了`"".Simpler.Add`方法，应该是新版本的Go编译器为了节省接口调用的开销而做的优化。**接下来我们稍微改下代码：

```go
package main

type Calculator interface {
	Add(a, b int) int
	Sub(a, b int) int
}

type Simpler struct {
	A int64
	B int32
}

//go:noinline
func (s Simpler) Add(a, b int) int {
	return a + b
}

//go:noinline
func (s Simpler) Sub(a, b int) int {
	return a - b
}

func CallAdd(c Calculator, a, b int) int {
	return c.Add(a, b)
}

func main() {
	var c Calculator = Simpler{A: 1234, B: 5678}
	CallAdd(c, 77, 88)
}
```

接下来我们直接分析 main 方法和 CallAdd 都做了什么

```asm
// func main()
"".main STEXT size=175 args=0x0 locals=0x50 funcid=0x0
	0x0000 00000 (main.go:27)	TEXT	"".main(SB), ABIInternal, $80-0

    // 检查是否需要栈扩展
	0x0000 00000 (main.go:27)	MOVQ	(TLS), CX
	0x0009 00009 (main.go:27)	CMPQ	SP, 16(CX)
	0x000d 00013 (main.go:27)	JLS	165

    // 准备 main 函数的栈空间
	0x0013 00019 (main.go:27)	SUBQ	$80, SP
	0x0017 00023 (main.go:27)	MOVQ	BP, 72(SP)
	0x001c 00028 (main.go:27)	LEAQ	72(SP), BP

    // 初始化 Simpler 对象
	0x0021 00033 (main.go:28)	MOVQ	$0, ""..autotmp_1+40(SP)
	0x002a 00042 (main.go:28)	MOVL	$0, ""..autotmp_1+48(SP)
	0x0032 00050 (main.go:28)	MOVQ	$1234, ""..autotmp_1+40(SP)
	0x003b 00059 (main.go:28)	MOVL	$5678, ""..autotmp_1+48(SP)

    // 通过调用 runtime.convT2Inoptr 的方式创建一个接口变量
	0x0043 00067 (main.go:28)	LEAQ	go.itab."".Simpler,"".Calculator(SB), AX
	0x004a 00074 (main.go:28)	MOVQ	AX, (SP)                    // 第一个参数为接口对应实现的 itab 偏移
	0x004e 00078 (main.go:28)	LEAQ	""..autotmp_1+40(SP), AX
	0x0053 00083 (main.go:28)	MOVQ	AX, 8(SP)                   // 第二个参数为绑定对象的指针(地址)
	// runtime/iface.go
	//
	// func convT2Inoptr(tab *itab, elem unsafe.Pointer) (i iface)
	0x0058 00088 (main.go:28)	CALL	runtime.convT2Inoptr(SB)

	// 从返回值中初始化接口变量 c
	0x005d 00093 (main.go:28)	MOVQ	16(SP), AX      // 返回值的 iface.tab 字段
	0x0062 00098 (main.go:28)	MOVQ	24(SP), CX      // 返回值的 iface.data 字段
	0x0067 00103 (main.go:28)	MOVQ	AX, "".c+56(SP) // 使用返回值初始化 c.tab 字段
	0x006c 00108 (main.go:28)	MOVQ	CX, "".c+64(SP) // 使用返回值初始化 c.data 字段

	// 准备调用 CallAdd 的参数
	0x0071 00113 (main.go:29)	MOVQ	"".c+56(SP), AX // 准备参数 c 的 tab 字段
	0x0076 00118 (main.go:29)	MOVQ	"".c+64(SP), CX // 准备参数 c 的 data 字段
	0x007b 00123 (main.go:29)	MOVQ	AX, (SP)    // 初始化参数 c 的 tab 字段
	0x007f 00127 (main.go:29)	MOVQ	CX, 8(SP)   // 初始化参数 c 的 data 字段
	0x0084 00132 (main.go:29)	MOVQ	$77, 16(SP) // CallAdd 的第二个参数
	0x008d 00141 (main.go:29)	MOVQ	$88, 24(SP) // CallAdd 的第三个参数
	0x0096 00150 (main.go:29)	CALL	"".CallAdd(SB)

    // 回收 main 函数的栈空间
	0x009b 00155 (main.go:30)	MOVQ	72(SP), BP
	0x00a0 00160 (main.go:30)	ADDQ	$80, SP
	0x00a4 00164 (main.go:30)	RET

    // 执行栈扩展方法, 扩展完成后返回函数入口再次检查
	0x00a5 00165 (main.go:30)	NOP
	0x00a5 00165 (main.go:27)	CALL	runtime.morestack_noctxt(SB)
	0x00aa 00170 (main.go:27)	JMP	0
```

关于以上内容我们需要关注几个点：

1、`LEAQ`指令用于获取一个符号或者内存的地址，`go.itab."".Simpler,"".Calculator(SB)`这个符号表示的 `Simpler` 类型在 `Calculator` 接口上对应的 `itab` 数据结构相对基址。

2、由于我们将会在其他函数中使用对象 `Simpler`，所以这个对象会逃逸到堆上。方法 `runtime.convT2Inoptr` 的作用就是在堆上创建相对应的 `iface` 数据结构，而 `iface` 数据结构就是实现接口动态调用的关键。`convT2Inoptr` 方法的正确断句是 `conv/T2I/noptr`，其表达的意思将类型转换为接口且其中不含指针。与之相对应的还有 `runtime.convT2I` 方法。对象是否逃逸我们可以使用 `tool compile -m -l main.go` 来进行检查（`-m` 参数可以使用多次）

```bash
$ go tool compile -m -l main.go
main.go:23:14: leaking param: c
main.go:28:6: Simpler{...} escapes to heap		// 从这里可以发现 Simpler 对象逃逸到堆上了
...
```

接下来来看看 `CallAdd` 方法是如何在 `iface` 基础上实现动态调用的

```asm
// func CallAdd(c Calculator, a, b int) int
"".CallAdd STEXT size=112 args=0x28 locals=0x30 funcid=0x0
	0x0000 00000 (main.go:23)	TEXT	"".CallAdd(SB), ABIInternal, $48-40
	// 栈扩展
	0x0000 00000 (main.go:23)	MOVQ	(TLS), CX
	0x0009 00009 (main.go:23)	CMPQ	SP, 16(CX)
	0x000d 00013 (main.go:23)	JLS	105

	// 开辟栈空间
	0x000f 00015 (main.go:23)	SUBQ	$48, SP
	0x0013 00019 (main.go:23)	MOVQ	BP, 40(SP)
	0x0018 00024 (main.go:23)	LEAQ	40(SP), BP

	// 给返回值赋零值初始化
	0x001d 00029 (main.go:23)	MOVQ	$0, "".~r3+88(SP)

	// 实现动态调用
	0x0026 00038 (main.go:24)	MOVQ	"".c+56(SP), AX     // 获取接口变量 c.tab 指针的值
	0x002b 00043 (main.go:24)	TESTB	AL, (AX)            // 检查 itab.interfacetype 指针是否为空
	0x002d 00045 (main.go:24)	MOVQ	"".a+72(SP), CX     // 获取参数 a 的值
	0x0032 00050 (main.go:24)	MOVQ	"".b+80(SP), DX     // 获取参数 b 的值
	0x0037 00055 (main.go:24)	MOVQ	24(AX), AX          // 获取 itab.fun 函数指针的值
	//
	// type itab struct {
    //     inter *interfacetype     // offset=0, size = 8
    //     _type *_type             // offset=8, size = 8
    //     hash  uint32             // offset=16, size = 4
    //     _     [4]byte            // offset=20, size = 4
    //     fun   [1]uintptr         // offset=24, size = variable
    // }
	//
	0x003b 00059 (main.go:24)	MOVQ	"".c+64(SP), BX     // 获取接口变量 c.data 的值
	0x0040 00064 (main.go:24)	MOVQ	BX, (SP)            // 第一个参数 c.data
	0x0044 00068 (main.go:24)	MOVQ	CX, 8(SP)           // 第二个参数 a
	0x0049 00073 (main.go:24)	MOVQ	DX, 16(SP)          // 第三个参数 b
	0x004e 00078 (main.go:24)	CALL	AX                  // itab.fun(c.data, a, b)
	0x0050 00080 (main.go:24)	MOVQ	24(SP), AX          // 获取函数的返回值
	0x0055 00085 (main.go:24)	MOVQ	AX, ""..autotmp_4+32(SP)    // 将返回值赋给一个临时变量
	0x005a 00090 (main.go:24)	MOVQ	AX, "".~r3+88(SP)           // 为 CallAdd 的返回值赋值

	// 回收栈空间并返回
	0x005f 00095 (main.go:24)	MOVQ	40(SP), BP
	0x0064 00100 (main.go:24)	ADDQ	$48, SP
	0x0068 00104 (main.go:24)	RET

	// 回收栈空间
	0x0069 00105 (main.go:24)	NOP
	0x0069 00105 (main.go:23)	CALL	runtime.morestack_noctxt(SB)
	0x006e 00110 (main.go:23)	JMP	0
```

同样，我们来看以上需要关注的几个点

1、接口的动态调用依靠的是 `itab` 结构中的 `fun` 字段中所保存的函数指针实现。

2、所有的动态调用过程中编译器都将接收者为值类型的方法转换为指针接收者，这种行为了是为了方便调用和优化。所以 `itab.func` 的第一个参数（接收者）的值为 `c.data` 指针（接口绑定类型实例的指针）

#### 总结

接口的动态调用依赖于 `iface.tab` 中的 `fun` 字段所保存的函数指针实现。接口的动态调用分为两个步骤

1、构建 `iface` 数据结构，一般在接口变量初始化时完成

2、**如果上下文可以直接推断类型，则直接调用具体类型的方法实现。否则通过接口变量中的函数指针调用接口绑定实例的方法**。



### 接口调用代价

在接口的动态调用过程中存在两不分多余损耗，第一是接口实例化的过程（也就是创建iface数据结构）。另一部分是接口的方法调用，这是一个函数指针的间接调用（动态计算后的跳转调用），这对现代计算机CPU的执行不是非常友好（会导致CPU缓存失效和分支预测失败，这也会导致一部分的性能损失）。

我们可以通过如下代码来对比测试，看看接口动态调用的性能损失到底有多大：

```go
//
// wear_test.go
//
//  go test -bench="Bench" -cpu=1 -count=5 wear_test.go
//

package wear

import (
	"testing"
)

type Identifier interface {
	Inline() int32
	NoInline() int32
}

type ID struct {
	id int32
}

func (id *ID) Inline() int32 {
	return id.id
}

//go:noinline
func (id *ID) NoInline() int32 {
	return id.id
}

//
// reflect/value.go
//
// Dummy annotation marking that the value x escapes,
// for use in cases where the reflect code is so clever that
// the compiler cannot follow.
func escapes(x interface{}) {
	if dummy.b {
		dummy.x = x
	}
}

var dummy struct {
	b bool
	x interface{}
}

//go:noinline
func DirectInline(id *ID) int32 {
	return id.Inline()
}

//go:noinline
func DirectNoInline(id *ID) int32 {
	return id.NoInline()
}

//go:noinline
func InterfaceInline(id Identifier) int32 {
	return id.Inline()
}

//go:noinline
func InterfaceNoInline(id Identifier) int32 {
	return id.NoInline()
}

func BenchmarkID_Direct(b *testing.B) {
	var ret int32

	b.Run("noinline", func(b *testing.B) {
		x := &ID{id: 1234}
		escapes(x)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ret = DirectNoInline(x)
		}
	})
	b.Run("inline", func(b *testing.B) {
		x := &ID{id: 1234}
		escapes(x)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ret = DirectInline(x)
		}
	})

	_ = ret
}

func BenchmarkID_Interface(b *testing.B) {
	var ret int32

	b.Run("noinline", func(b *testing.B) {
		var x Identifier = &ID{id: 1234}
		escapes(x)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ret = InterfaceNoInline(x)
		}
	})
	b.Run("inline", func(b *testing.B) {
		var x Identifier = &ID{id: 1234}
		escapes(x)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ret = InterfaceInline(x)
		}
	})

	_ = ret
}
```

以下分别是 `go1.16`，`go1.17`，`go1.18` 得到的结果

```text
//
// go1.16
//
goos: linux
goarch: amd64
cpu: Intel(R) Xeon(R) CPU E5-2683 v4 @ 2.10GHz
BenchmarkID_Direct/noinline         	279827311	         4.230 ns/op
BenchmarkID_Direct/noinline         	284898583	         4.198 ns/op
BenchmarkID_Direct/noinline         	285709496	         4.204 ns/op
BenchmarkID_Direct/noinline         	285689091	         4.201 ns/op
BenchmarkID_Direct/noinline         	285710798	         4.202 ns/op
BenchmarkID_Direct/inline           	502259485	         2.392 ns/op
BenchmarkID_Direct/inline           	495132288	         2.388 ns/op
BenchmarkID_Direct/inline           	495564354	         2.390 ns/op
BenchmarkID_Direct/inline           	502272045	         2.390 ns/op
BenchmarkID_Direct/inline           	502392055	         2.390 ns/op
BenchmarkID_Interface/noinline      	266717086	         4.513 ns/op
BenchmarkID_Interface/noinline      	266016693	         4.504 ns/op
BenchmarkID_Interface/noinline      	266364024	         4.515 ns/op
BenchmarkID_Interface/noinline      	263780900	         4.503 ns/op
BenchmarkID_Interface/noinline      	266884749	         4.501 ns/op
BenchmarkID_Interface/inline        	265814242	         4.497 ns/op
BenchmarkID_Interface/inline        	266684151	         4.499 ns/op
BenchmarkID_Interface/inline        	266325372	         4.539 ns/op
BenchmarkID_Interface/inline        	266335770	         4.524 ns/op
BenchmarkID_Interface/inline        	264714562	         4.511 ns/op
PASS


//
// go1.17
//
goos: linux
goarch: amd64
cpu: Intel(R) Xeon(R) CPU E5-2683 v4 @ 2.10GHz
BenchmarkID_Direct/noinline         	368764939	         3.224 ns/op
BenchmarkID_Direct/noinline         	378324794	         3.168 ns/op
BenchmarkID_Direct/noinline         	378442860	         3.165 ns/op
BenchmarkID_Direct/noinline         	378489505	         3.167 ns/op
BenchmarkID_Direct/noinline         	378627369	         3.169 ns/op
BenchmarkID_Direct/inline           	562518901	         2.137 ns/op
BenchmarkID_Direct/inline           	562454290	         2.139 ns/op
BenchmarkID_Direct/inline           	564468903	         2.123 ns/op
BenchmarkID_Direct/inline           	566852649	         2.118 ns/op
BenchmarkID_Direct/inline           	568746331	         2.117 ns/op
BenchmarkID_Interface/noinline      	343047410	         3.491 ns/op
BenchmarkID_Interface/noinline      	342796851	         3.494 ns/op
BenchmarkID_Interface/noinline      	343993393	         3.508 ns/op
BenchmarkID_Interface/noinline      	343607748	         3.485 ns/op
BenchmarkID_Interface/noinline      	344529655	         3.484 ns/op
BenchmarkID_Interface/inline        	344531984	         3.476 ns/op
BenchmarkID_Interface/inline        	344546524	         3.479 ns/op
BenchmarkID_Interface/inline        	345821228	         3.476 ns/op
BenchmarkID_Interface/inline        	345530970	         3.472 ns/op
BenchmarkID_Interface/inline        	342848203	         3.485 ns/op
PASS


//
// go1.18
//
goos: linux
goarch: amd64
cpu: Intel(R) Xeon(R) CPU E5-2683 v4 @ 2.10GHz
BenchmarkID_Direct/noinline         	382653601	         3.130 ns/op
BenchmarkID_Direct/noinline         	384281214	         3.122 ns/op
BenchmarkID_Direct/noinline         	384861854	         3.119 ns/op
BenchmarkID_Direct/noinline         	387765333	         3.110 ns/op
BenchmarkID_Direct/noinline         	387017601	         3.102 ns/op
BenchmarkID_Direct/inline           	576926984	         2.084 ns/op
BenchmarkID_Direct/inline           	576976302	         2.081 ns/op
BenchmarkID_Direct/inline           	577804605	         2.085 ns/op
BenchmarkID_Direct/inline           	576790406	         2.077 ns/op
BenchmarkID_Direct/inline           	577854663	         2.086 ns/op
BenchmarkID_Interface/noinline      	347317480	         3.482 ns/op
BenchmarkID_Interface/noinline      	346113297	         3.463 ns/op
BenchmarkID_Interface/noinline      	345904214	         3.476 ns/op
BenchmarkID_Interface/noinline      	345881388	         3.474 ns/op
BenchmarkID_Interface/noinline      	345325503	         3.481 ns/op
BenchmarkID_Interface/inline        	345897781	         3.480 ns/op
BenchmarkID_Interface/inline        	344225434	         3.493 ns/op
BenchmarkID_Interface/inline        	339639932	         3.485 ns/op
BenchmarkID_Interface/inline        	344404767	         3.488 ns/op
BenchmarkID_Interface/inline        	344211804	         3.487 ns/op
PASS
```

通过对比我们可以发现，通过接口进行动态调用会有大概 2ns 的性能损失，而且还是在这种简单方法上（无形中放大了接口调用的耗时），如果方法中带有复杂的逻辑计算，则真实的**性能损失基本可以忽略不计**。

从不同版本的测试来看，Go1.17之后得益于寄存器的调用约定也进一步减少了接口的动态调用损耗，且官方团队也在不断地进行优化，所以除非是纳秒必争的项目，不然可以忽略接口动态调用带来的性能损耗。



### 空接口数据结构

空接口（`interface{}`）是没有任何方法集的接口，所以空接口内部不需要维护和动态内存分配相关的数据结构 `itab`。空接口只关心其中存放的具体类型是什么，具体类型的值是什么。所以空接口的底层数据结构也非常简单：

```go
//
// runtime2.go
//
type eface struct {
	_type *_type
	data  unsafe.Pointer
}
```

从 `eface` 数据结构看起来，空接口也不是真的为空，其中保存了绑定实例的类型和指针拷贝。所以即使绑定的实例为空，空接口也不为空（类型不为空）。

由于空接口没有方法集，所以空接口变量实例化后的真正用途不是借口方法的动态调用。而是在Go是实现多态的支持：

* 通过接口类型断言
* 通过接口类型查询
* 通过反射
