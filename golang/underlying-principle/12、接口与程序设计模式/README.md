接口与程序设计模式
-----------------------------

**隐藏细节**：接口可以对对象进行必要的抽象，调用方只需要满足接口的调用方式，就可以使用实现方提供的强大功能，而不必关注对方具体的实现细节。

控制系统复杂性：通过接口，我们能够以模块化的方式构建起复杂、庞大的系统。

**权限控制**：如果调用方不满足接口的规范，就无法与之进行沟通。因此系统可以通过接口控制接入方式和接入方行为，降低安全风险。



### Go 语言中的接口

在 Go 语言中，没有任何形式的基于类型的继承，取而代之的是使用接口实现扁平化、面向组合的设计模式。在 Go 语言中，接口是一种特殊的类型，其是其他类型可以实现的方法签名的集合（方法签名只包含方法名，输入参数和返回值）。

在 Go 语言中，接口包含两种形式：一种是带方法签名的接口，一种是空接口。



#### 接口的实现

在 Go 语言中，接口的实现的隐式的。即我们不用明确地指出某一个类型实现了一个接口，只要在某一类型的方法中实现了接口中的全部方法签名，就意味着此类型实现了这一接口。



#### 接口的动态调用

接口的动态调用的过程实质上是调用当前接口动态类型中具体方法的过程。同时在对接口变量进行动态调用时，调用的方法只能是接口中具有的方法。



### 接口底层原理

当把具体的类型赋值给接口时，需要判断该类型是否实现接口中的所有方法。Go 语言在编译时对对具体类型和接口中的方法进行相同规则的排序，再对接口中的方法进行比较。

在编译时，查找类型是否实现接口的逻辑位于 `implements` 。其中通过遍历接口列表，并与类型方法列表中对应的位置进行比较，判断类型是否实现了接口：

```go
//
// cmd/compile/internal/typecheck/subr.go
//

func implements(t, iface *types.Type, m, samename **types.Field, ptr *int) bool {
	t0 := t
	if t == nil {
		return false
	}

	if t.IsInterface() {
		i := 0
		tms := t.AllMethods().Slice()
		for _, im := range iface.AllMethods().Slice() {
			for i < len(tms) && tms[i].Sym != im.Sym {
				i++
			}
			if i == len(tms) {
				*m = im
				*samename = nil
				*ptr = 0
				return false
			}
			tm := tms[i]
			if !types.Identical(tm.Type, im.Type) {
				*m = im
				*samename = tm
				*ptr = 0
				return false
			}
		}

		return true
	}

	t = types.ReceiverBaseType(t)
	var tms []*types.Field
	if t != nil {
		CalcMethods(t)
		tms = t.AllMethods().Slice()
	}
	i := 0
	for _, im := range iface.AllMethods().Slice() {
		if im.Broke() {
			continue
		}
		for i < len(tms) && tms[i].Sym != im.Sym {
			i++
		}
		if i == len(tms) {
			*m = im
			*samename, _ = ifacelookdot(im.Sym, t, true)
			*ptr = 0
			return false
		}
		tm := tms[i]
		if tm.Nointerface() || !types.Identical(tm.Type, im.Type) {
			*m = im
			*samename = tm
			*ptr = 0
			return false
		}
		followptr := tm.Embedded == 2

		// if pointer receiver in method,
		// the method does not exist for value types.
		rcvr := tm.Type.Recv().Type
		if rcvr.IsPtr() && !t0.IsPtr() && !followptr && !types.IsInterfaceMethod(tm.Type) {
			if false && base.Flag.LowerR != 0 {
				base.Errorf("interface pointer mismatch")
			}

			*m = im
			*samename = nil
			*ptr = 1
			return false
		}
	}

	return true
}
```



#### 接口组成

接口也是 Go 语言中的一种类型，带方法前面的接口在运行时的具体结构由 `iface` 构成：

```go
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

type interfacetype struct {
	typ     _type
	pkgpath name
	mhdr    []imethod
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
```

在 `iface` 结构中  `data` 字段储存了接口动态类型实例的指针，而 `tab` 字段中保存着接口的类型、接口中的动态数据类型、动态数据类型的函数指针等。

在 `itab` 结构中 `_type` 字段代表接口存储的动态类型，而 `inter` 中保存的是接口本身的类型。

在 `_type` 结构体中，`hash` 字段是接口动态类型的唯一标识，最后的 `fun` 字段中保存着接口动态类型中的函数指针列表，用于运行时接口动态调用类型方法。



#### 接口内存逃逸分析

存储在接口中的值必须能够获取其地址（被保存在 `data` 字段中），所以平时分配在栈中的值一旦赋值给接口之后，会发生内存逃逸，实际上会在堆区为其分配内存。



#### 接口动态调用过程

接口的动态调用过程可以通过检查其反汇编代码查看，其基本思路是——先找到接口的位置，再通过偏移量找到要调用的函数指针，最后准备参数并进行调用。
