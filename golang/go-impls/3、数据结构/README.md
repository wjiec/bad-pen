数据结构
-------------

数据结构和算法是程序中最重要的两个组成部分。



### 数组

数组是一种数据结构，时相同类型元素的集合。计算机会为数组分配一块连续的内存来保存其中的元素。在 Go 中，我们通常会从两个维度来描述数组——数组中存储的元素类型和数组最多能存储的元素个数。

Go 语言的数组在初始化之后大小就无法改变，且类型元素相同但大小不同的数组类型在 Go 中是完全不同的。



#### 初始化

Go 语言的数组有两种创建方式，一种是显式指定数组大小，一种是使用 `[...]Type` 让编译器在编译时自动计算大小：

```go
var arr1 [3]int
var arr2 [...]int{1,2,3}
```

当我们使用 `[...]T` 创建数组时，编译器会在 `cmd/compile/internal/gc.typecheckcomplit` 函数中推推导数组的大小。

#### 语句转换

编译器会根据元素数量的不同分别做对应的优化：

* 当元素少于或等于 4 个时，会直接将数组中的元素放置到栈上
* 当元素多于 4 个时，会将数组中的元素放置到静态区并在运行时取出

```go
//
// src/cmd/compile/internal/walk/complit.go
//
func anylit(n ir.Node, var_ ir.Node, init *ir.Nodes) {
	t := n.Type()
	switch n.Op() {
	case ir.OARRAYLIT:
		n := n.(*ir.CompLitExpr)
		if !t.IsStruct() && !t.IsArray() {
			base.Fatalf("anylit: not struct/array")
		}

		if isSimpleName(var_) && len(n.List) > 4 {
			// lay out static data
			vstat := readonlystaticname(t)

			ctxt := inInitFunction
			if n.Op() == ir.OARRAYLIT {
				ctxt = inNonInitFunction
			}
			fixedlit(ctxt, initKindStatic, n, vstat, init)

			// copy static to var
			appendWalkStmt(init, ir.NewAssignStmt(base.Pos, var_, vstat))

			// add expressions to automatic
			fixedlit(inInitFunction, initKindDynamic, n, var_, init)
			break
		}

		var components int64
		if n.Op() == ir.OARRAYLIT {
			components = t.NumElem()
		} else {
			components = int64(t.NumFields())
		}
		// initialization of an array or struct with unspecified components (missing fields or arrays)
		if isSimpleName(var_) || int64(len(n.List)) < components {
			appendWalkStmt(init, ir.NewAssignStmt(base.Pos, var_, nil))
		}

		fixedlit(inInitFunction, initKindLocalCode, n, var_, init)
	}
}
```

#### 访问与赋值

Go 语言中可以通过编译期间的静态类型检查判断数组是否越界。Go 语言对数组的访问有比较多的检查，它不仅会在编译期间提前发现一些简单的越界错误并插入用于检测数组上限的函数调用，还会在运行期间通过插入的函数保证不会发生越界。

```go
//
// GOSSAFUNC=ArrayBoundCheck go build array.go
//

func ArrayBoundCheck() int {
	var arr [3]int
	i := 4
	v := arr[i]
	return v
}
```



### 切片

在日常开发中，更常用的数据结构是切片，即动态数组，其长度不固定，我们可以向切片中追加元素，它会在容量不足时自动扩容。

#### 数据结构

在编译期间切片的时 `cmd/compile/internal/types.Slice` 类型，但是在运行时切片可以由 `reflect.SliceHeader` 结构体表示。

```go
//
// src/reflect/value.go
//

// SliceHeader is the runtime representation of a slice.
// It cannot be used safely or portably and its representation may
// change in a later release.
// Moreover, the Data field is not sufficient to guarantee the data
// it references will not be garbage collected, so programs must keep
// a separate, correctly typed pointer to the underlying data.
//
// In new code, use unsafe.Slice or unsafe.SliceData instead.
type SliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}
```

我们可以将切片理解成一块连续的内存空间加上长度与容量的标识。

#### 初始化

切片的初始化有三种形式：

* 使用下标在数组或切片之前创建：不会复制底层数组，只是对原有数据结构的一个「视图」
* 字面量形式创建：在编译期会展开为创建在数组之上的一个「视图」
* 使用 make 关键字创建：如果切片发生逃逸或者非常大，则会在运行时通过 `runtime.makeslice` 来创建。否则会直接回滚到字面量的形式在编译期间创建。

#### 追加和扩容

使用 append 关键字向切片中追加元素也是常见的切片操作。当切片容量不足时，运行时会调用 `runtime.growslice` 函数为切片扩容。



### 哈希表



