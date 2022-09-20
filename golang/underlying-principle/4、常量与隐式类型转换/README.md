常量与隐式类型转换
----------------------------

Go 语言最独特的功能之一是对常量的处理。Go 语言规范中的常量规则是精心设计的并且是语言特有的，其在编译时为静态类型的 Go 语言提供了灵活性，以使编写的代码更具可读性和直观性，同时保持类型安全。

在 Go 语言中使用 `const` 关键字来声明常量，在声明时可以指定或忽略类型。如下所示：

```go
const UntypedInteger = 12345
const TypedInteger int64 = 12345
```

其中，等式左边的常量叫做命名常量，等式右边的常量叫做未命名常量，拥有未定义的类型。未命名常量只会在编译期间存在，因此其不会存储在内存中，而命名常量存在于内存静态只读区，不能被修改。同时 Go 语言禁止对常量取地址的操作。



### 常量的隐式类型转换

在 Go 语言中，变量之间没有隐式类型转换，不同的类型之间只能强制转换（**底层类型相同的命名类型与字面量之间无需强制类型转换**）。但是编译器可以进行变量与常量之间的隐式类型转换。常量（包括命名常量与未命名常量，即字面量）的隐式类型转换主要遵循以下规则：

1. 如果常量使用与整数兼容的类型，也可以将浮点常量隐式转换为整数变量
2. 编译器可以在整数常量与 float64 变量之间进行隐式转换
3. 除移位操作外，如果操作数两边是不同的类型的未命名常量，则结果类型的优先级为：`int < rune < float < imag`
4. 常量与具体类型的变量之间的运算，会使用已有的具体类型

对以上规则，我们有如下的代码示例：

```go
const (
	Integer       = 12345
	Float         = 1.234
	IntCompatible = 123.0
)

func main() {
	var i int = IntCompatible // 1
	fmt.Printf("typeof i = %T\n", i)
	fmt.Printf("typeof IntCompatible = %T\n", IntCompatible)
	// typeof i = int
	// typeof IntCompatible = float64

	var f float64 = Integer // 2
	fmt.Printf("typeof f = %T\n", f)
	fmt.Printf("typeof Integer = %T\n", Integer)
	// typeof f = float64
	// typeof Integer = int

	p := Float * Integer // 3
	fmt.Printf("typeof p = %T\n", p)
	// typeof p = float64

	s := Integer * time.Second
	fmt.Printf("typeof s = %T\n", s)
	// typeof s = time.Duration
}
```



### 常量隐式类型转换的原理

常量以及常量具有的一系列隐式类型转换需要借助 Go 语言编译器完成。对于设计常量的运算，统一在编译时的类型检查阶段完成，由 `cmd/compile/internal/typecheck/const.go:defaultlit2` 函数完成统一处理。

需要注意的是，并不是所有类型组合都能进行隐式转换，例如字符串不能和非字符串进行组合，布尔类型与 nil 不能与其他类型进行组合。
