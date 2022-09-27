数组
------

Go 语言中的数组与其他语言中的额数组有显著不同的特征。例如，其不能进行扩容、在复制和传递时为值复制。人们通常将数组与 Go 语言中另一个重要结构——切片进行对比。



### 数组的声明

数组在内存中是一片连续的区域，需要在初始化时指定其长度，数组的大小取决于数组中存放的元素的大小。需要注意的是**数组的长度也是数组类型的一部分**，不同类型数组（元素类型或长度不同）之间不能进行比较。而切片只能与 nil 进行比较。



#### 数组的访问

我们只能访问数组中已有的元素，如果数组访问越界，则在编译时（对于数字常量或命名常量）或运行时（变量下标）会报错。



#### 数组的值复制

Go 语言中的数组在赋值和函数调用时的形参都是值复制。



### 数组底层原理

数组在编译时构建抽象语法树阶段的数据类型为 TARRAY，通过 NewArray 函数进行创建，AST 节点的 Op 操作为 OARRAYLIT。其中 Array 内部的数据结构存储了数组中元素的类型及数组的大小

```go
//
// go/types/type.go
//

// An Array represents an array type.
type Array struct {
	len  int64
	elem Type
}

// NewArray returns a new array type for the given element type and length.
// A negative length indicates an unknown length.
func NewArray(elem Type, len int64) *Array { return &Array{len: len, elem: elem} }
```



#### 数组字面量初始化原理

数组字面量的初始化在编译时类型检查阶段进行，通过 typecheckarraylit 函数循环字面量并分别进行赋值：

```go
//
// cmd/compile/internal/typecheck/typecheck.go
//

// typecheckarraylit type-checks a sequence of slice/array literal elements.
func typecheckarraylit(elemType *types.Type, bound int64, elts []ir.Node, ctx string) int64 {
	// If there are key/value pairs, create a map to keep seen
	// keys so we can check for duplicate indices.
	var indices map[int64]bool
	for _, elt := range elts {
		if elt.Op() == ir.OKEY {
			indices = make(map[int64]bool)
			break
		}
	}

	var key, length int64
	for i, elt := range elts {
		ir.SetPos(elt)
		r := elts[i]
		var kv *ir.KeyExpr
		if elt.Op() == ir.OKEY {
			elt := elt.(*ir.KeyExpr)
			elt.Key = Expr(elt.Key)
			key = IndexConst(elt.Key)
			if key < 0 {
				if !elt.Key.Diag() {
					if key == -2 {
						base.Errorf("index too large")
					} else {
						base.Errorf("index must be non-negative integer constant")
					}
					elt.Key.SetDiag(true)
				}
				key = -(1 << 30) // stay negative for a while
			}
			kv = elt
			r = elt.Value
		}

		r = pushtype(r, elemType)
		r = Expr(r)
		r = AssignConv(r, elemType, ctx)
		if kv != nil {
			kv.Value = r
		} else {
			elts[i] = r
		}

		if key >= 0 {
			if indices != nil {
				if indices[key] {
					base.Errorf("duplicate index in %s: %d", ctx, key)
				} else {
					indices[key] = true
				}
			}

			if bound >= 0 && key >= bound {
				base.Errorf("array index %d out of bounds [0:%d]", key, bound)
				bound = -1
			}
		}

		key++
		if key > length {
			length = key
		}
	}

	return length
}
```



#### 数组字面量编译时内存优化

数组在编译时还会进行重要的优化，在函数 walk 遍历阶段，anylit 函数用于处理各种类型的字面量。当数组的长度小于 4 时，在运行时数组会被放到栈中，当数组长度大于 4 时，数组会被放置到内存的静态只读区。fixedlit 函数将执行数组初始化和赋值的逻辑：

```go
//
// cmd/compile/internal/walk/complit.go
//

func anylit(n ir.Node, var_ ir.Node, init *ir.Nodes) {
	t := n.Type()
	switch n.Op() {
	case ir.OSTRUCTLIT, ir.OARRAYLIT:
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



#### 数组索引与访问越界原理

Go 语言中对越界的判断有一部分是在编译期间类型检查阶段完成的，typecheck1 函数会对访问数组的索引进行验证：

```go
// typecheck1 should ONLY be called from typecheck.
func typecheck1(n ir.Node, top int) ir.Node {
	if n, ok := n.(*ir.Name); ok {
		typecheckdef(n)
	}

	switch n.Op() {
	case ir.OINDEX:
		n := n.(*ir.IndexExpr)
		return tcIndex(n)
	}

	// No return n here!
	// Individual cases can type-assert n, introducing a new one.
	// Each must execute its own return n.
}

// tcIndex typechecks an OINDEX node.
func tcIndex(n *ir.IndexExpr) ir.Node {
	n.X = Expr(n.X)
	n.X = DefaultLit(n.X, nil)
	n.X = implicitstar(n.X)
	l := n.X
	n.Index = Expr(n.Index)
	r := n.Index
	t := l.Type()
	if t == nil || r.Type() == nil {
		n.SetType(nil)
		return n
	}
	switch t.Kind() {
	default:
		base.Errorf("invalid operation: %v (type %v does not support indexing)", n, t)
		n.SetType(nil)
		return n

	case types.TSTRING, types.TARRAY, types.TSLICE:
		n.Index = indexlit(n.Index)
		if t.IsString() {
			n.SetType(types.ByteType)
		} else {
			n.SetType(t.Elem())
		}
		why := "string"
		if t.IsArray() {
			why = "array"
		} else if t.IsSlice() {
			why = "slice"
		}

		if n.Index.Type() != nil && !n.Index.Type().IsInteger() {
			base.Errorf("non-integer %s index %v", why, n.Index)
			return n
		}

		if !n.Bounded() && ir.IsConst(n.Index, constant.Int) {
			x := n.Index.Val()
			if constant.Sign(x) < 0 {
				base.Errorf("invalid %s index %v (index must be non-negative)", why, n.Index)
			} else if t.IsArray() && constant.Compare(x, token.GEQ, constant.MakeInt64(t.NumElem())) {
				base.Errorf("invalid array index %v (out of bounds for %d-element array)", n.Index, t.NumElem())
			} else if ir.IsConst(n.X, constant.String) && constant.Compare(x, token.GEQ, constant.MakeInt64(int64(len(ir.StringVal(n.X))))) {
				base.Errorf("invalid string index %v (out of bounds for %d-byte string)", n.Index, len(ir.StringVal(n.X)))
			} else if ir.ConstOverflow(x, types.Types[types.TINT]) {
				base.Errorf("invalid %s index %v (index too large)", why, n.Index)
			}
		}

	case types.TMAP:
		n.Index = AssignConv(n.Index, t.Key(), "map index")
		n.SetType(t.Elem())
		n.SetOp(ir.OINDEXMAP)
		n.Assigned = false
	}
	return n
}
```

Go 语言运行时在发现数组、切片和字符串的越界操作时，会由运行时的 panicIndex 和 runtime.goPanicIndex 函数触发程序的运行时错误并导致崩溃。



