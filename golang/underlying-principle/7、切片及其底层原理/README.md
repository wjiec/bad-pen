切片使用方法与底层原理
----------------------------------

在 Go 语言中，切片是长度可变的序列，序列中的每个元素都有相同的类型。和数组不同的是，切片不用指定固定长度。一个切片在运行时由指针（data）、长度（len）和容量（cap）这三部分组成：

```go
type SliceHeader {
    Data uintptr
    Len int
    Cap int
}
```

其中，指针部分指向切片元素对应底层数组元素的地址，长度对应切片中元素的数量（不能超过容量），而容量一般是底层数组的长度。



#### 切片的使用

在只声明而不初始化时，切片的值为 nil，切片的初始化需要使用内置的 make 函数，或者通过切片字面量的方式进行声明和初始化。

```go
var s1 []int // nil
var s2 []int = make([]int, 2) // len = 2, cap = 2
var s3 []int = make([]int, 3, 5) // len = 3, cap = 5
var s4 = []int{1,2,3,4,5} // len = 5, cap = 5
```

和数组一样，切片中的数据也是内存中的一片连续的区域。要获取切片某一区域的连续数组，可以通过下表的方式对切片进行截断：

```go
var s = []int{0,1,2,3,4,5,6,7,8,9}

s1 := s[2:4] // [2,3] len = 2, cap = 8
s2 := s[:5] // [0,1,2,3,4] len = 5, cap = 10
s3 := s[5:] // [5,6,7,8,9] len = 5, cap = 5
s4 := s[:5:5] // [0,1,2,3,4] len = 5, cap = 5
s5 := s[5:9:10] // [5 6 7 8] len = 4, cap = 5
```

需要注意的是，在截取切片之后切片的底层数组依然指向原始切片的底层数组的开始位置。同时在 Go 语言中，参数都是以值传递的方式传递，所以传递切片实际上只会拷贝 `SliceHeader` 结构体，并不会拷贝底层数组。

```go
func dump(s []int) {
	fmt.Printf("dump :: &s.Data = %#x\n", (*reflect.SliceHeader)(unsafe.Pointer(&s)).Data)
}

func main() {
	s := []int{1, 2, 3, 4, 5}
	fmt.Printf("main :: &s.Data = %#x\n", (*reflect.SliceHeader)(unsafe.Pointer(&s)).Data)

	dump(s)
}

// main :: &s.Data = 0xc000074ef8
// dump :: &s.Data = 0xc000074ef8
```

Go 语言内置的 append 函数可以添加新的元素到切片的末尾，它可以接受可变长度的元素，并且可以自动对切片进行扩容。



### 切片的底层原理

在编译时构建抽象语法树阶段会将切片构建为如下类型：

```go
// A Slice represents a slice type.
type Slice struct {
	elem Type
}

// NewSlice returns a new slice type for the given element type.
func NewSlice(elem Type) *Slice { return &Slice{elem: elem} }
```

编译时使用 `NewSlice` 函数新建一个切片类型，并需要传递切片元素的类型。



#### 字面量初始化

当使用形如 `[]int{1, 2, 3}` 的字面量初始化新的切片时，会创建一个数组并保存在静态区中，并在堆区创建一个新的切片，在程序启动时将静态区的数据复制到堆区中，这样可以加快切片的初始化速度：

```go
//
// cmd/compile/internal/walk/complit.go
//

func slicelit(ctxt initContext, n *ir.CompLitExpr, var_ ir.Node, init *ir.Nodes) {
	// make an array type corresponding the number of elements we have
	t := types.NewArray(n.Type().Elem(), n.Len)
	types.CalcSize(t)

	if ctxt == inNonInitFunction {
		// put everything into static array
		vstat := staticinit.StaticName(t)

		fixedlit(ctxt, initKindStatic, n, vstat, init)
		fixedlit(ctxt, initKindDynamic, n, vstat, init)

		// copy static to slice
		var_ = typecheck.AssignExpr(var_)
		name, offset, ok := staticinit.StaticLoc(var_)
		if !ok || name.Class != ir.PEXTERN {
			base.Fatalf("slicelit: %v", var_)
		}
		staticdata.InitSlice(name, offset, vstat.Linksym(), t.NumElem())
		return
	}

	// recipe for var = []t{...}
	// 1. make a static array
	//	var vstat [...]t
	// 2. assign (data statements) the constant part
	//	vstat = constpart{}
	// 3. make an auto pointer to array and allocate heap to it
	//	var vauto *[...]t = new([...]t)
	// 4. copy the static array to the auto array
	//	*vauto = vstat
	// 5. for each dynamic part assign to the array
	//	vauto[i] = dynamic part
	// 6. assign slice of allocated heap to var
	//	var = vauto[:]
	//
	// an optimization is done if there is no constant part
	//	3. var vauto *[...]t = new([...]t)
	//	5. vauto[i] = dynamic part
	//	6. var = vauto[:]

	// if the literal contains constants,
	// make static initialized array (1),(2)
	var vstat ir.Node

	mode := getdyn(n, true)
	if mode&initConst != 0 && !isSmallSliceLit(n) {
		if ctxt == inInitFunction {
			vstat = readonlystaticname(t)
		} else {
			vstat = staticinit.StaticName(t)
		}
		fixedlit(ctxt, initKindStatic, n, vstat, init)
	}

	// make new auto *array (3 declare)
	vauto := typecheck.Temp(types.NewPtr(t))

	// set auto to point at new temp or heap (3 assign)
	var a ir.Node
	if x := n.Prealloc; x != nil {
		// temp allocated during order.go for dddarg
		if !types.Identical(t, x.Type()) {
			panic("dotdotdot base type does not match order's assigned type")
		}
		a = initStackTemp(init, x, vstat)
	} else if n.Esc() == ir.EscNone {
		a = initStackTemp(init, typecheck.Temp(t), vstat)
	} else {
		a = ir.NewUnaryExpr(base.Pos, ir.ONEW, ir.TypeNode(t))
	}
	appendWalkStmt(init, ir.NewAssignStmt(base.Pos, vauto, a))

	if vstat != nil && n.Prealloc == nil && n.Esc() != ir.EscNone {
		// If we allocated on the heap with ONEW, copy the static to the
		// heap (4). We skip this for stack temporaries, because
		// initStackTemp already handled the copy.
		a = ir.NewStarExpr(base.Pos, vauto)
		appendWalkStmt(init, ir.NewAssignStmt(base.Pos, a, vstat))
	}

	// put dynamics into array (5)
	var index int64
	for _, value := range n.List {
		if value.Op() == ir.OKEY {
			kv := value.(*ir.KeyExpr)
			index = typecheck.IndexConst(kv.Key)
			if index < 0 {
				base.Fatalf("slicelit: invalid index %v", kv.Key)
			}
			value = kv.Value
		}
		a := ir.NewIndexExpr(base.Pos, vauto, ir.NewInt(index))
		a.SetBounded(true)
		index++

		// TODO need to check bounds?

		switch value.Op() {
		case ir.OSLICELIT:
			break

		case ir.OARRAYLIT, ir.OSTRUCTLIT:
			value := value.(*ir.CompLitExpr)
			k := initKindDynamic
			if vstat == nil {
				// Generate both static and dynamic initializations.
				// See issue #31987.
				k = initKindLocalCode
			}
			fixedlit(ctxt, k, value, a, init)
			continue
		}

		if vstat != nil && ir.IsConstNode(value) { // already set by copy from static value
			continue
		}

		// build list of vauto[c] = expr
		ir.SetPos(value)
		as := typecheck.Stmt(ir.NewAssignStmt(base.Pos, a, value))
		as = orderStmtInPlace(as, map[string][]*ir.Name{})
		as = walkStmt(as)
		init.Append(as)
	}

	// make slice out of heap (6)
	a = ir.NewAssignStmt(base.Pos, var_, ir.NewSliceExpr(base.Pos, ir.OSLICE, vauto, nil, nil, nil))

	a = typecheck.Stmt(a)
	a = orderStmtInPlace(a, map[string][]*ir.Name{})
	a = walkStmt(a)
	init.Append(a)
}
```



#### make 初始化

当使用形如 `make([]int, 3, 4)` 的方式初始化切片时，在类型检查阶段`typecheck1` 函数中，如下所示：

```go
// typecheck1 should ONLY be called from typecheck.
func typecheck1(n ir.Node, top int) ir.Node {
    switch n.Op() {
	case ir.OTSLICE:
		n := n.(*ir.SliceType)
		return tcSliceType(n)
    }
}

// tcSliceType typechecks an OTSLICE node.
func tcSliceType(n *ir.SliceType) ir.Node {
	n.Elem = typecheckNtype(n.Elem)
	if n.Elem.Type() == nil {
		return n
	}
	t := types.NewSlice(n.Elem.Type())
	n.SetOTYPE(t)
	types.CheckSize(t)
	return n
}

// tcSliceType typechecks an OTSLICE node.
func tcSliceType(n *ir.SliceType) ir.Node {
	n.Elem = typecheckNtype(n.Elem)
	if n.Elem.Type() == nil {
		return n
	}
	t := types.NewSlice(n.Elem.Type())
	n.SetOTYPE(t)
	types.CheckSize(t)
	return n
}
```

编译时堆字面量额重要优化是判断变量应该被分配到栈中还是应该逃逸到堆中。如果 make 函数初始化了一个太大的切片，则该切片会逃逸到堆中。如果分配了一个较小的欺骗，则会直接在栈中分配。此临界值定义在 `cmd/compile/internal/ir/cfg.go` 文件中的 `MaxImplicitStackVarSize` 变量中，默认大小为 64K。



#### 切片的复制

我们可以使用 copy 函数实现切片的复制。在运行时 copy 函数会调用 memmove 函数，用于实现内存的复制。如果使用协程调用的方式 `go copy(dst, src)` 或加入 race 检测，则会在运行时调用 `slicestringcopy` 或 `slicecopy` 函数，进行额外的检查。
