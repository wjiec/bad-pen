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

设计哈希表的关键点——哈希函数和冲突解决方法。其中，哈希函数在很大程度上能够决定哈希表的读写性能。而哈希冲突问题无法避免，常见的方法是开放地址法和拉链法。

* 开放地址法：如果发生了冲突，则会将键值对写入下一个索引不为空的位置
* 拉链法：拉链法会使用链表数组作为哈希底层的数据结构，我们可以将它看成会扩展的二维数组。

#### 数据结构

Go 语言运行时同时使用多个数据结构组合来表示哈希表，其中 `runtime.hmap` 是最核心的结构体：

```go
//
// runtime/map.go
//

// A header for a Go map.
type hmap struct {
	// Note: the format of the hmap is also encoded in cmd/compile/internal/reflectdata/reflect.go.
	// Make sure this stays in sync with the compiler's definition.
	count     int // # live cells == size of map.  Must be first (used by len() builtin)
	flags     uint8
	B         uint8  // log_2 of # of buckets (can hold up to loadFactor * 2^B items)
	noverflow uint16 // approximate number of overflow buckets; see incrnoverflow for details
	hash0     uint32 // hash seed

	buckets    unsafe.Pointer // array of 2^B Buckets. may be nil if count==0.
	oldbuckets unsafe.Pointer // previous bucket array of half the size, non-nil only when growing
	nevacuate  uintptr        // progress counter for evacuation (buckets less than this have been evacuated)

	extra *mapextra // optional fields
}

// mapextra holds fields that are not present on all maps.
type mapextra struct {
	// If both key and elem do not contain pointers and are inline, then we mark bucket
	// type as containing no pointers. This avoids scanning such maps.
	// However, bmap.overflow is a pointer. In order to keep overflow buckets
	// alive, we store pointers to all overflow buckets in hmap.extra.overflow and hmap.extra.oldoverflow.
	// overflow and oldoverflow are only used if key and elem do not contain pointers.
	// overflow contains overflow buckets for hmap.buckets.
	// oldoverflow contains overflow buckets for hmap.oldbuckets.
	// The indirection allows to store a pointer to the slice in hiter.
	overflow    *[]*bmap
	oldoverflow *[]*bmap

	// nextOverflow holds a pointer to a free overflow bucket.
	nextOverflow *bmap
}

// A bucket for a Go map.
type bmap struct {
	// tophash generally contains the top byte of the hash value
	// for each key in this bucket. If tophash[0] < minTopHash,
	// tophash[0] is a bucket evacuation state instead.
	tophash [bucketCnt]uint8
	// Followed by bucketCnt keys and then bucketCnt elems.
	// NOTE: packing all the keys together and then all the elems together makes the
	// code a bit more complicated than alternating key/elem/key/elem/... but it allows
	// us to eliminate padding which would be needed for, e.g., map[int64]int8.
	// Followed by an overflow pointer.
}
```

其中 `runtime.bmap` 就是所谓的「桶」，每一个桶可以存储 8 个键值对。溢出桶是 Go 语言还使用 C 语言实现时的设计，由于它能降低扩容频率而沿用至今。

`runtime.bmap` 中的其他字段在运行时也都是通过计算内存地址的方式进行访问的。所以它的定义中不包含这些字段，不过我们可以根据编译期间的 `cmd/compile/internal/reflectdata/reflect.MapBucketType` 函数获得完整的结构：

```go
type bmap struct {
  // field = append(field, makefield("topbits", arr))
  topbits [8]uint8
  
  // keys := makefield("keys", arr)
  keys [8]keytype
  
  // elems := makefield("elems", arr)
  elems [8]elemtype
  
  // overflow := makefield("overflow", otyp)
  overflow uintptr
}
```

#### 初始化

哈希表的初始化支持使用字面量或者通过运行时 `make` 的方式进行。

* 字面量形式：当哈希表的元素数量小于等于 25 个时，编译器会将字面量初始化的结构体转换为逐个加入哈希表的形式。超过 25 个时，将会创建两个数组分别存储键和值，这些键值对会通过循环加入哈希表
* 运行时：如果哈希表在栈上切元素数量小于 8 个时，将会使用小容量的哈希表优化创建。否则都会使用 `runtime.makemap` 来创建。
  * 当桶的数量小于 2 ^ 4 （16）时将不会创建溢出桶，否则会创建 2 ^ (B - 4) 个溢出桶。

#### 读写操作

在编译的类型检查期间，`map[key]` 以及类似的读取操作都会被转换成哈希表的 `OINDEXMAP` 操作，中间代码生成阶段会将这些 `OINDEXMAP` 操作转换成 `runtime.mapaccess1` 或 `mapaccess2` 方法。

* 当接收一个参数时，会使用 `runtime.mapaccess1`，该函数仅会返回一个指向目标值的指针
* 当接收两个参数时，会使用 `runtime.mapaccess2`，除了返回目标值，还会返回一个用于表示当前键是否存在的布尔值

在查找键过程中，用于选择桶序号的时哈希的最低几位，而用于加速访问的是哈希的高 8 位，这种设计能够降低同一个桶中有大量相等 tophash 的概率以免影响性能。

当 `map[key] = val` 形式出现时，该表达式会在编译期间转换成 `runtime.mapassign` 函数的调用。

##### 哈希表扩容

随着哈希表中元素的逐渐增加，哈希表性能会逐渐恶化，所以需要更多的桶和更大的内存保证哈希表的读写性能。`runtime.mapassign` 会在以下两种情况发生时触发扩容：

* 装载因子超过 6.5：触发双倍扩容
* 哈希表使用了过多的溢出桶：触发等量扩容

哈希表的数据迁移是在 `runtime.evacuate` 中完成的，它会对传入桶中的元素进行再分配。

##### 哈希表删除

在编译期间，哈希表的删除操作会被转换成 `runtime.mapdelete` 中的一个：`runtime.mapdelete`、`runtime.mapdelete_faststr`、`runtime.mapdelete_fast32`、`runtime.mapdelete_fast64`。用于处理删除逻辑的函数与哈希表的 `runtime.mapassign` 几乎完全相同。



### 字符串

