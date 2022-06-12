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



### inject库

Github：[codegangsta/inject](https://github.com/codegangsta/inject)

inject 是 Go 语言依赖注入的实现，它实现了对结构体和函数的依赖注入。



#### inject 原理解释

```go
// Package inject provides utilities for mapping and injecting dependencies in various ways.
package inject

import (
	"fmt"
	"reflect"
)

// Injector represents an interface for mapping and injecting dependencies into structs
// and function arguments.
type Injector interface {
	Applicator
	Invoker
	TypeMapper
	// SetParent sets the parent of the injector. If the injector cannot find a
	// dependency in its Type map it will check its parent before returning an
	// error.
	SetParent(Injector)
}

// Applicator represents an interface for mapping dependencies to a struct.
type Applicator interface {
	// Maps dependencies in the Type map to each field in the struct
	// that is tagged with 'inject'. Returns an error if the injection
	// fails.
	Apply(interface{}) error
}

// Invoker represents an interface for calling functions via reflection.
type Invoker interface {
	// Invoke attempts to call the interface{} provided as a function,
	// providing dependencies for function arguments based on Type. Returns
	// a slice of reflect.Value representing the returned values of the function.
	// Returns an error if the injection fails.
	Invoke(interface{}) ([]reflect.Value, error)
}

// TypeMapper represents an interface for mapping interface{} values based on type.
type TypeMapper interface {
	// Maps the interface{} value based on its immediate type from reflect.TypeOf.
	Map(interface{}) TypeMapper
	// Maps the interface{} value based on the pointer of an Interface provided.
	// This is really only useful for mapping a value as an interface, as interfaces
	// cannot at this time be referenced directly without a pointer.
	MapTo(interface{}, interface{}) TypeMapper
	// Provides a possibility to directly insert a mapping based on type and value.
	// This makes it possible to directly map type arguments not possible to instantiate
	// with reflect like unidirectional channels.
	Set(reflect.Type, reflect.Value) TypeMapper
	// Returns the Value that is mapped to the current type. Returns a zeroed Value if
	// the Type has not been mapped.
	Get(reflect.Type) reflect.Value
}

type injector struct {
	values map[reflect.Type]reflect.Value
	parent Injector
}

// InterfaceOf dereferences a pointer to an Interface type.
// It panics if value is not an pointer to an interface.
func InterfaceOf(value interface{}) reflect.Type {
	t := reflect.TypeOf(value)

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Interface {
		panic("Called inject.InterfaceOf with a value that is not a pointer to an interface. (*MyInterface)(nil)")
	}

	return t
}

// New returns a new Injector.
func New() Injector {
	return &injector{
		values: make(map[reflect.Type]reflect.Value),
	}
}

// Invoke attempts to call the interface{} provided as a function,
// providing dependencies for function arguments based on Type.
// Returns a slice of reflect.Value representing the returned values of the function.
// Returns an error if the injection fails.
// It panics if f is not a function
func (inj *injector) Invoke(f interface{}) ([]reflect.Value, error) {
	t := reflect.TypeOf(f)

	var in = make([]reflect.Value, t.NumIn()) //Panic if t is not kind of Func
	for i := 0; i < t.NumIn(); i++ {
		argType := t.In(i)
		val := inj.Get(argType)
		if !val.IsValid() {
			return nil, fmt.Errorf("Value not found for type %v", argType)
		}

		in[i] = val
	}

	return reflect.ValueOf(f).Call(in), nil
}

// Maps dependencies in the Type map to each field in the struct
// that is tagged with 'inject'.
// Returns an error if the injection fails.
func (inj *injector) Apply(val interface{}) error {
	v := reflect.ValueOf(val)

	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil // Should not panic here ?
	}

	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		structField := t.Field(i)
		if f.CanSet() && (structField.Tag == "inject" || structField.Tag.Get("inject") != "") {
			ft := f.Type()
			v := inj.Get(ft)
			if !v.IsValid() {
				return fmt.Errorf("Value not found for type %v", ft)
			}

			f.Set(v)
		}

	}

	return nil
}

// Maps the concrete value of val to its dynamic type using reflect.TypeOf,
// It returns the TypeMapper registered in.
func (i *injector) Map(val interface{}) TypeMapper {
	i.values[reflect.TypeOf(val)] = reflect.ValueOf(val)
	return i
}

func (i *injector) MapTo(val interface{}, ifacePtr interface{}) TypeMapper {
	i.values[InterfaceOf(ifacePtr)] = reflect.ValueOf(val)
	return i
}

// Maps the given reflect.Type to the given reflect.Value and returns
// the Typemapper the mapping has been registered in.
func (i *injector) Set(typ reflect.Type, val reflect.Value) TypeMapper {
	i.values[typ] = val
	return i
}

func (i *injector) Get(t reflect.Type) reflect.Value {
	val := i.values[t]

	if val.IsValid() {
		return val
	}

	// no concrete types found, try to find implementors
	// if t is an interface
	if t.Kind() == reflect.Interface {
		for k, v := range i.values {
			if k.Implements(t) {
				val = v
				break
			}
		}
	}

	// Still no type found, try to look it up on the parent
	if !val.IsValid() && i.parent != nil {
		val = i.parent.Get(t)
	}

	return val

}

func (i *injector) SetParent(parent Injector) {
	i.parent = parent
}
```

1、通过 `inject.New` 创建一个注入引擎，返回一个实现了 `Inkjector` 接口的内部实例

2、调用 `TypeMapper` 接口的方法注入结构体的字段值或函数的实参值

3、调用 `Invoke` 方法执行被注入的函数，或者调用 `Applicator` 接口方法获得被注入后的结构实例



### 反射的优缺点

1、在库或框架内部使用反射，而不是把反射接口暴露给调用者；将复杂性留在内部，简单性放到接口

2、框架代码才考虑使用反射，一般的业务代码没必要抽象到反射的层次，这种过度设计会带来复杂度的提升

3、除非没有其他办法，否则不要使用反射技术



#### 反射的优点

1、通用性：库或框架需要一种通用的处理模式，可以借助反射极大地简化设计，而不是针对每一种场景做硬编码处理

2、灵活性：反射提供了一种程序了解自己和改变自己的能力，这为一些测试工具的开发提供了有力的支持



#### 反射的缺点

1、反射是脆弱的：反射可以在运行时修改程序的状态，这种修改没有经过编译器的严格检查，不正确的处理很容易造成程序等额崩溃

2、反射是晦涩难懂的：由于反射涉及语言的运行时，没有具体的类型系统的约束，接口的抽象级别较高但实现复杂，导致反射代码难以理解

3、反射有部分性能损失：反射提供动态修改程序状态的能力，必然不是直接的地址引用，而是要借助运行时构建一个抽象层，这种间接返回会带来性能损失
