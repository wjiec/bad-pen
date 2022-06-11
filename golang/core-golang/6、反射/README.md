反射
------

在计算机科学中，反射是指计算机程序在运行时（runtime）可以访问、检测和修改自身状态或行为的一种能力。Go 语言反射实现的基础是编译器和运行时把类型信息以合适的数据结构保存在可执行文件中。



### 基本概念

Go 反射基础是接口和类型系统。Go 巧妙地借助了示例到接口的转换过程中所生成的数据结构 `eface` （空接口的内部结构），从而基于这个转换后的数据结构访问和操作实例的值和类型。

#### 基本数据结构和入口函数

在反射包中有两个基本数据结构：`Type` 和 `Value` ，每个类型都有一个自己的构造函数，其他所有方法和衍生类型都是在此之上建立的。

##### reflect.Type

`rtype` 是反射包中用于描述类型通用公共信息的结构体。这个 `rtype` 其实和 `runtime._type` 结构体是一个东西，只是因为包的隔离性从而分开定义。其中结构中的字段都是描述类型的通用信息，同时为每一种基础类型都封装了一个特定的结构

```go
//
// reflect/type.go
//

// rtype is the common implementation of most values.
// It is embedded in other struct types.
//
// rtype must be kept in sync with ../runtime/type.go:/^type._type.
type rtype struct {
	size       uintptr
	ptrdata    uintptr // number of bytes in the type that can contain pointers
	hash       uint32  // hash of type; avoids computation in hash tables
	tflag      tflag   // extra type information flags
	align      uint8   // alignment of variable with this type
	fieldAlign uint8   // alignment of struct field with this type
	kind       uint8   // enumeration for C
	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	equal     func(unsafe.Pointer, unsafe.Pointer) bool
	gcdata    *byte   // garbage collection data
	str       nameOff // string form
	ptrToThis typeOff // type for pointer to this type, may be zero
}

// arrayType represents a fixed array type.
type arrayType struct {
	rtype
	elem  *rtype // array element type
	slice *rtype // slice type
	len   uintptr
}

// chanType represents a channel type.
type chanType struct {
	rtype
	elem *rtype  // channel element type
	dir  uintptr // channel direction (ChanDir)
}

// funcType represents a function type.
//
// A *rtype for each in and out parameter is stored in an array that
// directly follows the funcType (and possibly its uncommonType). So
// a function type with one method, one input, and one output is:
//
//	struct {
//		funcType
//		uncommonType
//		[2]*rtype    // [0] is in, [1] is out
//	}
type funcType struct {
	rtype
	inCount  uint16
	outCount uint16 // top bit is set if last input parameter is ...
}

// interfaceType represents an interface type.
type interfaceType struct {
	rtype
	pkgPath name      // import path
	methods []imethod // sorted by hash
}

// mapType represents a map type.
type mapType struct {
	rtype
	key    *rtype // map key type
	elem   *rtype // map element (value) type
	bucket *rtype // internal bucket structure
	// function for hashing keys (ptr to key, seed) -> hash
	hasher     func(unsafe.Pointer, uintptr) uintptr
	keysize    uint8  // size of key slot
	valuesize  uint8  // size of value slot
	bucketsize uint16 // size of bucket
	flags      uint32
}

// ptrType represents a pointer type.
type ptrType struct {
	rtype
	elem *rtype // pointer element (pointed at) type
}

// sliceType represents a slice type.
type sliceType struct {
	rtype
	elem *rtype // slice element type
}

// structType represents a struct type.
type structType struct {
	rtype
	pkgPath name
	fields  []structField // sorted by offset
}
```

我们可以通过 `relect.TypeOf` 来获取一个 `reflect.Type` 类型的接口变量，然后通过接口变量来获取对象的类型信息。接口中有以下方法：

* 所有类型通用的方法：以下这些是所有类型上都有的方法（仅介绍常用的）
  * `Name`：返回带包名的类型名称，**未命名类型则返回空字符串**
  * `Kind`：返回该类型的底层基础类型（`int, float64, map, slice, struct`等）
* 不同基础类型的专有方法，使用错误则会引发 panic（仅介绍常用的）
  * `Elem`：返回元素的类型（适用于Array、Chan、Map、Slice）和指针所指向的类型（Ptr）
  * `*Field*`：返回结构体的字段相关信息（适用于 Struct）
  * `*In*, *Out*`：输入输出参数相关信息（适用于 Function）

在使用 `reflect.TypeOf` 过程中根据参数的不同，返回的类型信息也有所不同：

```go
type Integer int

func (i Integer) String() string {
	return strconv.Itoa(int(i))
}

func main() {
	var i = 10
	var I Integer = 20
	fmt.Printf("(typeof i).Name() = %s\n", reflect.TypeOf(i).Name())
	fmt.Printf("(typeof I).Name() = %s\n", reflect.TypeOf(I).Name())
	// (typeof i).Name() = int
	// (typeof I).Name() = Integer

	fmt.Printf("(typeof i).Kind() = %s\n", reflect.TypeOf(i).Kind())
	fmt.Printf("(typeof I).Kind() = %s\n", reflect.TypeOf(I).Kind())
	fmt.Printf("(typeof i) == (typeof I) ? => %v\n", reflect.TypeOf(i) == reflect.TypeOf(I))
	// (typeof i).Kind() = int
	// (typeof I).Kind() = int
	// (typeof i) == (typeof I) ? => false

	e := new(fmt.Stringer)
	var v fmt.Stringer = I
	fmt.Printf("(typeof *e).Name() = %s\n", reflect.TypeOf(e).Elem().Name())
	fmt.Printf("(typeof v).Name() = %s\n", reflect.TypeOf(v).Name())
	// (typeof e).Name() = Stringer
	// (typeof v).Name() = Integer

	fmt.Printf("(typeof *e).Kind() = %s\n", reflect.TypeOf(e).Elem().Kind())
	fmt.Printf("(typeof v).Kind() = %s\n", reflect.TypeOf(v).Kind())
	// (typeof *e).Kind() = interface
	// (typeof v).Kind() = int
}
```

由以上代码，我们可以得出结论：

* 如果实参是一个具体类型变量，则 `reflct.TypeOf` 返回的就是具体类型的信息
* 如果实参是一个接口类型变量：
  * 接口变量绑定了具体的类型实例，则返回的是接口的动态类型（即所绑定的具体类型）信息
  * 是一个空接口（未绑定具体类型实例），则返回是接口自身的静态类型信息

##### reflect.Value

在反射包中，`reflect.Value` 是一个结构体，提供了一系列的方法给调用者

```go
//
// reflect/value.go
//

// Value is the reflection interface to a Go value.
//
// Not all methods apply to all kinds of values. Restrictions,
// if any, are noted in the documentation for each method.
// Use the Kind method to find out the kind of value before
// calling kind-specific methods. Calling a method
// inappropriate to the kind of type causes a run time panic.
//
// The zero Value represents no value.
// Its IsValid method returns false, its Kind method returns Invalid,
// its String method returns "<invalid Value>", and all other methods panic.
// Most functions and methods never return an invalid value.
// If one does, its documentation states the conditions explicitly.
//
// A Value can be used concurrently by multiple goroutines provided that
// the underlying Go value can be used concurrently for the equivalent
// direct operations.
//
// To compare two Values, compare the results of the Interface method.
// Using == on two Values does not compare the underlying values
// they represent.
type Value struct {
	// typ holds the type of the value represented by a Value.
	typ *rtype // 值的类型信息

	// Pointer-valued data or, if flagIndir is set, pointer to data.
	// Valid when either flagIndir is set or typ.pointers() is true.
	ptr unsafe.Pointer // 指向值的指针

	// flag holds metadata about the value.
	// The lowest bits are flag bits:
	//	- flagStickyRO: obtained via unexported not embedded field, so read-only
	//	- flagEmbedRO: obtained via unexported embedded field, so read-only
	//	- flagIndir: val holds a pointer to the data
	//	- flagAddr: v.CanAddr is true (implies flagIndir)
	//	- flagMethod: v is a method value.
	// The next five bits give the Kind of the value.
	// This repeats typ.Kind() except for method values.
	// The remaining 23+ bits give a method number for method values.
	// If flag.kind() != Func, code can assume that flagMethod is unset.
	// If ifaceIndir(typ), code can assume that flagIndir is set.
	flag // 标记字段

	// A method value represents a curried method invocation
	// like r.Read for some receiver r. The typ+val+flag bits describe
	// the receiver r, but the flag's Kind bits say Func (methods are
	// functions), and the top bits of the flag give the method number
	// in r's type's method table.
}
```

与获取类型类似，我们可以使用方法 `reflect.ValuOf` 来获得一个 `reflect.Value` 类型的实例。



#### 基础类型

`Type` 接口上有一个 `Kind()` 方法，返回的是一个整数枚举值，不同的值代表不同的类型（只是一个抽象的概念，并不是一个“类型”）。这个类型是根据编译器、运行时构建类型的内部数据结构的不同来划分的，不同的基础类型，其构建的最终内部数据结构不一样。在 `reflect` 包中，总共定义了以下类型枚举值：

```go
type Kind uint

const (
	Invalid Kind = iota
	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	Array
	Chan
	Func
	Interface
	Map
	Ptr
	Slice
	String
	Struct
	UnsafePointer
)
```

##### 底层类型与基础类型

底层类型是针对每一个具体的类型定义的，而”基础类型“只是一个抽象的概念仅用于区分不同的类型



### 反射规则

反射对象 `Value`，`Type` 和实例之间的相互转换 API 如下所示：



#### 从实例到 Value 对象

通过实例获得 `Value` 对象，可以直接使用 `reflect.ValueOf` 方法：

```go
func ValueOf(i interface{}) Value
```



#### 从实例到 Type 对象

通过实例获得 `Type` 对象，可以直接使用 `reflect.TypeOf` 方法：

```go
func TypeOf(i interface{}) Type
```



#### 从 Type 对象到 Value 对象

由于 `Type` 中只有类型信息，所以直接从一个 `Type` 对象里无法直接获得实例的 `Value` 对象，但是可以通过在 `Type` 对象上构建一个新的 `Value` 对象

```go
// 通过 Type 返回一个 Value, 该 Value 的类型是 *Type
func New(typ Type) Value

// 返回一个 Type 类型的零值，返回的 Value 对象即不可寻址也不可修改
func Zero(typ Type) Value
```

如果知道 `Type` 类型值的地址，则还有一个函数可以根据 `Type` 和该地址恢复出一个 `Value` 对象

```go
// 将指针 p 所指向的位置的内存解释为 Type 类型
func NewAt(typ Type, p unsafe.Pointer) Value
```



#### 从 Value 对象到 Type 对象

因为 `Value` 对象内部保存有 `Type` 类型的指针，所以我们可以通过如下方法获得 `Type` 对象

```go
func (v Value) Type() Type
```



#### 从 Value 对象到实例

由于 `Value` 对象中包含对象的类型和值信息，所以在 `Value` 对象上提供了丰富的方法来实现到实例的转换

```go
// 将 Value 对象转换为一个空接口实例，之后可以使用接口断言或者接口查询来进行转换
func (v Value) Interface() (i interface{})


func (v Value) Int() int64
func (v Value) Float() float64
func (v Value) Bool() bool
```



#### 从 Value 指针对象到 Value 值对象

从一个指针类型的 `Value` 对象获得值类型的 `Value` 对象，可以使用如下方法

```go
// 错误的调用该方法将会导致 panic
func (v Value) Elem() Value

// 不会引发 panic
func Indirect(v Value) Value {
	if v.Kind() != Ptr {
		return v
	}
	return v.Elem()
}
```



#### 指针类型的 Type 对象和值类型的 Type

在 `reflect` 包中有以下方法可以实现不同类型 `Type` 对象的转换

```go
type Type interface {
    // 从指针类型的 Type 到值类型的 Type
    Elem() Type
}

// 从值类型的 Type 到指针类型的 Type
func PtrTo(t Type) Type
```



#### Value 对象的可修改性

对于 `Value` 对象的修改涉及以下方法：

```go
// 返回 Value 对象是否可修改
func (v Value) CanSet() bool

// 修改 Value 对象的内容
func (v Value) Set(x Value)
```

如果我们调用 `reflect.ValueOf` 时传入的是一个值类型实例，则获得的 `Value` 对象实际上是执行原对象的副本，那么这个 `Value` 对象就是不可修改的。如果传入的是一个指针，虽然 `Value` 对象获得的也是一个指针副本，但是我们可以通过指针修改到原始对象，所以这个 `Value` 对象就是可修改的。

```go
type User struct {
	Name string
	Age  int
}

func main() {
	user := User{Name: "hello", Age: 18}

	rv := reflect.ValueOf(user)
	rp := reflect.ValueOf(&user)

	fmt.Printf("rv.CanSet() = %v\n", rv.CanSet())
	fmt.Printf("rp.CanSet() = %v\n", rp.Elem().CanSet())
	// rv.CanSet() = false
	// rp.CanSet() = true

	fmt.Printf("user = %+v\n", user)
	// user = {Name:hello Age:18}

	rp.Elem().FieldByName("Name").Set(reflect.ValueOf("world"))
	fmt.Printf("user = %+v\n", user)
	// user = {Name:world Age:18}
}
```

