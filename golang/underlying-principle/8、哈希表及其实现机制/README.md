哈希表与Go语言实现机制
------------------------------------

哈希表的原理是将多个键值对分散存储在 Bucket 中，给定一个键，哈希算法会计算出键的哈希值并将键值对存储在相应的桶中。



#### 哈希碰撞与解决方法

哈希函数在实际中遇到的最常见问题是哈希碰撞（Hash Collision），即不同的键通过哈希函数可能产生相同的哈希值。哈希碰撞导致同一个桶中可能存在多个元素，常见的避免哈希碰撞的方式有拉链法和开放寻址法。

* 拉链法将同一个桶中的元素通过链表的形式进行链接，随着桶中元素的增加，可以不断链接新的元素，同时不用预先为元素分配内存。拉链法的不足之处在于，需要存储额外的指针用于链接元素，这增加了整个哈希表的大小。同时由于链表存储的地址不连续，所以无法搞笑利用 CPU 告诉缓存。
* 开放寻址法在在插入新条目时，按照某种策略（例如直接向后查找）直到找到未使用的插槽为止。当搜索元素时，将按照相同的策略扫描，直到找到目标键值对或找到未使用的插槽为止。

Go 语言中的哈希表采用的是优化的拉链法，每一个桶中存储 8 个元素用于加速访问。



### map 的基本操作

在 Go 语言中，可以使用如下几种方式对哈希表进行声明或初始化：

```go
var m1 map[int]int
m1[1] = 111 // panic
fmt.Println(m1[2]) // 0

var m2 = make(map[int]int, count)

var m3 = map[int]int{
    1: -1,
    2: -2,
}
```

第二种声明方式是通过 make 函数进行初始化，其中 make 函数的第二个参数代表初始化创建 map 的长度，当不存在这个参数时，其默认长度为 0。



#### map 的操作

我们可以通过 delete 关键字来删除 map 中的键值对，多次对相同的 key 进行删除是安全的。

```go
func main() {
	var m = make(map[int]int)

	delete(m, 1)
	delete(m, 1)
	delete(m, 1)
}
```



#### map 的并发问题

在 Go 语言中，map 并不支持并发的读写，Go 语言仅支持并发读取 map。对此，官方的解释是：

>### Why are map operations not defined to be atomic?
>
>After long discussion it was decided that the typical use of maps did not require safe access from multiple goroutines, and in those cases where it did, the map was probably part of some larger data structure or computation that was already synchronized. Therefore requiring that all map operations grab a mutex would slow down most programs and add safety to few. This was not an easy decision, however, since it means uncontrolled map access can crash the program.
>
>The language does not preclude atomic map updates. When required, such as when hosting an untrusted program, the implementation could interlock map access.
>
>Map access is unsafe only when updates are occurring. As long as all goroutines are only reading—looking up elements in the map, including iterating through it using a `for` `range` loop—and not changing the map by assigning to elements or doing deletions, it is safe for them to access the map concurrently without synchronization.
>
>As an aid to correct map use, some implementations of the language contain a special check that automatically reports at run time when a map is modified unsafely by concurrent execution.



### 哈希表底层结构

在 Go 语言中，map 的底层实现如下所示：

```go
// A header for a Go map.
type hmap struct {
	// Note: the format of the hmap is also encoded in cmd/compile/internal/reflectdata/reflect.go.
	// Make sure this stays in sync with the compiler's definition.
    
    // map 中键值对的数量
	count     int // # live cells == size of map.  Must be first (used by len() builtin)
    
    // 当前 map 的状态（是否正在写入等）
	flags     uint8
    
    // 表示当前 map 中桶的数量为 2^b
	B         uint8  // log_2 of # of buckets (can hold up to loadFactor * 2^B items)
    
    // map 中溢出桶的数量，主要是为了避免溢出桶过大导致内存泄露
	noverflow uint16 // approximate number of overflow buckets; see incrnoverflow for details
    
    // 哈希函数的随机种子
	hash0     uint32 // hash seed

    // 当前 map 中的桶的指针
	buckets    unsafe.Pointer // array of 2^B Buckets. may be nil if count==0.
    
    // 扩容之后保存旧桶的执政，当所有数据前已完成后会清空这个值
	oldbuckets unsafe.Pointer // previous bucket array of half the size, non-nil only when growing
    
    // 在扩容时标记当前旧桶中小于 nevacuate 的数据以及转移到新桶中
	nevacuate  uintptr        // progress counter for evacuation (buckets less than this have been evacuated)

    // 存储 map 中的溢出桶
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
    
    // key [bucketCnt]K
    // value [bucketCnt]V
    // overflow unsafe.Pointer
}
```

代表桶的 bmap 结构在运行时只列出了首个字段 tophash，这是一个长度为 8 的数组，其中按顺序保存件的哈希值的前 8 位。



### 哈希表原理

Go 语言选择将 key/value 分开存储而不是以 key/value/key/value 的形式存储，主要是为了在字节对其时压缩空间。在进行访问操作时，首先找到桶的位置，由于 key/value 由于在编译时已经确定了其大小，所以之后可以在运行时通过指针操作就可以找到特定位置的元素。

```go
hash = hashfunc(key)
bucket = buckets[hash % len(buckets)]

for i := 0; i < 8; i++ {
    if bucket.tophash[i] == hash {
        return *(bucket + 8 + kSize * i), *(bucket + 8 + kSize * 8 + vSize * i)
    }
}
```



#### 溢出桶

在执行 `m[key] = value` 操作时，当指定的桶中存储的数据已满时，并不会字节开辟一个新桶，而是将数据放置到溢出桶中，每个桶的最后都存储了 overflow，即溢出桶的指针。同时，在执行查找操作时，如果 key 对应的哈希值在桶中找不到，那么还需要遍历溢出桶中的数据。

当创建 map 时，运行时回提前创建好一些溢出桶存储在 `extra` 字段中，这样当出现溢出时，可以用提前创建好的桶而不用提前申请额外的内存空间，只有在预分配的桶使用完了，才会新建溢出桶。

```go
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
```



#### map 的扩容

当发生以下两种情况时，map 会进行重建：

* map 超过了负载因子大小
* 溢出桶的数量过多

在哈希表中对负载因子的定义为：**负载因子 = 元素的数量 / 桶的数量**。

在 Go 语言中定义的负载因子是 6.5，当超过其大小时，map 会进行扩容，增大到旧表桶数量的两倍，同时旧桶中的数据会存到 oldbuckets 中，并在后续的操作中逐步分散转移到新桶中。

当 map 中的溢出桶太多时，这时 map 只会新建和旧表相同的桶数，目的是为了防止溢出桶的数量增长导致的内存泄露问题。



#### map 中的删除操作

当删除 map 中的一个键值对时，delete 操作会找到指定的桶，如果存在则释放 key/value 所引用的内存，同时在 tophash 中在指定位置存储 `emptyOne` ，表示当前位置是空的。在删除操作的同时运行时还会探测当前要删除的元素之后是否都是空的，如果是，则 tophash 中会存储为 `emptyRest`，这样之后的查找操作如果遇到 `emptyRest` 则可以直接退出。



### 深入哈希表原理

map 相关实现位于 `runtime/map.go` 文件中



#### make 初始化原理

在编译时的函数 walk 遍历阶段，会根据情况指定运行时应该调用 `runtime.makemap` 函数还是 `runtime.makemap64` 函数。

```go
//
// cmd/compile/internal/walk/builtin.go
//

// walkMakeMap walks an OMAKEMAP node.
func walkMakeMap(n *ir.MakeExpr, init *ir.Nodes) ir.Node {
	// Map initialization with a variable or large hint is
	// more complicated. We therefore generate a call to
	// runtime.makemap to initialize hmap and allocate the
	// map buckets.

	// When hint fits into int, use makemap instead of
	// makemap64, which is faster and shorter on 32 bit platforms.
	fnname := "makemap64"
	argtype := types.Types[types.TINT64]

	// Type checking guarantees that TIDEAL hint is positive and fits in an int.
	// See checkmake call in TMAP case of OMAKE case in OpSwitch in typecheck1 function.
	// The case of hint overflow when converting TUINT or TUINTPTR to TINT
	// will be handled by the negative range checks in makemap during runtime.
	if hint.Type().IsKind(types.TIDEAL) || hint.Type().Size() <= types.Types[types.TUINT].Size() {
		fnname = "makemap"
		argtype = types.Types[types.TINT]
	}

	fn := typecheck.LookupRuntime(fnname)
	fn = typecheck.SubstArgTypes(fn, hmapType, t.Key(), t.Elem())
	return mkcall1(fn, n.Type(), init, reflectdata.TypePtr(n.Type()), typecheck.Conv(hint, argtype), h)
}
```

其中，makemap64最后也会调用 makemap 函数，并保证创建 map 的长度不能超过 int 的大小（区别 32 位和 64 位的 int）

```go
//
// runtime/map.go
//

func makemap64(t *maptype, hint int64, h *hmap) *hmap {
	if int64(int(hint)) != hint {
		hint = 0
	}
	return makemap(t, int(hint), h)
}

// makemap implements Go map creation for make(map[k]v, hint).
// If the compiler has determined that the map or the first bucket
// can be created on the stack, h and/or bucket may be non-nil.
// If h != nil, the map can be created directly in h.
// If h.buckets != nil, bucket pointed to can be used as the first bucket.
func makemap(t *maptype, hint int, h *hmap) *hmap {
	mem, overflow := math.MulUintptr(uintptr(hint), t.bucket.size)
	if overflow || mem > maxAlloc {
		hint = 0
	}

	// initialize Hmap
	if h == nil {
		h = new(hmap)
	}
	h.hash0 = fastrand()

	// Find the size parameter B which will hold the requested # of elements.
	// For hint < 0 overLoadFactor returns false since hint < bucketCnt.
	B := uint8(0)
	for overLoadFactor(hint, B) {
		B++
	}
	h.B = B

	// allocate initial hash table
	// if B == 0, the buckets field is allocated lazily later (in mapassign)
	// If hint is large zeroing this memory could take a while.
	if h.B != 0 {
		var nextOverflow *bmap
		h.buckets, nextOverflow = makeBucketArray(t, h.B, nil)
		if nextOverflow != nil {
			h.extra = new(mapextra)
			h.extra.nextOverflow = nextOverflow
		}
	}

	return h
}
```

makemap 函数会计算出需要的桶的数量，即 `log2(N)`，并调用 `makeBucketArray` 函数生成桶和溢出桶。如果初始化时生成了溢出桶，则会放置到 extra 字段中。

```go
//
// runtime/map.go
//

// makeBucketArray initializes a backing array for map buckets.
// 1<<b is the minimum number of buckets to allocate.
// dirtyalloc should either be nil or a bucket array previously
// allocated by makeBucketArray with the same t and b parameters.
// If dirtyalloc is nil a new backing array will be alloced and
// otherwise dirtyalloc will be cleared and reused as backing array.
func makeBucketArray(t *maptype, b uint8, dirtyalloc unsafe.Pointer) (buckets unsafe.Pointer, nextOverflow *bmap) {
	base := bucketShift(b)
	nbuckets := base
	// For small b, overflow buckets are unlikely.
	// Avoid the overhead of the calculation.
	if b >= 4 {
		// Add on the estimated number of overflow buckets
		// required to insert the median number of elements
		// used with this value of b.
		nbuckets += bucketShift(b - 4)
		sz := t.bucket.size * nbuckets
		up := roundupsize(sz)
		if up != sz {
			nbuckets = up / t.bucket.size
		}
	}

	if dirtyalloc == nil {
		buckets = newarray(t.bucket, int(nbuckets))
	} else {
		// dirtyalloc was previously generated by
		// the above newarray(t.bucket, int(nbuckets))
		// but may not be empty.
		buckets = dirtyalloc
		size := t.bucket.size * nbuckets
		if t.bucket.ptrdata != 0 {
			memclrHasPointers(buckets, size)
		} else {
			memclrNoHeapPointers(buckets, size)
		}
	}

	if base != nbuckets {
		// We preallocated some overflow buckets.
		// To keep the overhead of tracking these overflow buckets to a minimum,
		// we use the convention that if a preallocated overflow bucket's overflow
		// pointer is nil, then there are more available by bumping the pointer.
		// We need a safe non-nil pointer for the last overflow bucket; just use buckets.
		nextOverflow = (*bmap)(add(buckets, base*uintptr(t.bucketsize)))
		last := (*bmap)(add(buckets, (nbuckets-1)*uintptr(t.bucketsize)))
		last.setoverflow(t, (*bmap)(buckets))
	}
	return buckets, nextOverflow
}
```



#### 字面量初始化原理

字面量的初始化最终也会被转换为 make 操作，map 的长度被自动推断为字面量的长度，其核心逻辑如下：

```go
//
// cmd/compile/internal/walk/complit.go
//

func maplit(n *ir.CompLitExpr, m ir.Node, init *ir.Nodes) {
	// make the map var
	a := ir.NewCallExpr(base.Pos, ir.OMAKE, nil, nil)
	a.SetEsc(n.Esc())
	a.Args = []ir.Node{ir.TypeNode(n.Type()), ir.NewInt(int64(len(n.List)))}
	litas(m, a, init)

	entries := n.List

	// The order pass already removed any dynamic (runtime-computed) entries.
	// All remaining entries are static. Double-check that.
	for _, r := range entries {
		r := r.(*ir.KeyExpr)
		if !isStaticCompositeLiteral(r.Key) || !isStaticCompositeLiteral(r.Value) {
			base.Fatalf("maplit: entry is not a literal: %v", r)
		}
	}

	if len(entries) > 25 {
		// For a large number of entries, put them in an array and loop.

		// build types [count]Tindex and [count]Tvalue
		tk := types.NewArray(n.Type().Key(), int64(len(entries)))
		te := types.NewArray(n.Type().Elem(), int64(len(entries)))

		tk.SetNoalg(true)
		te.SetNoalg(true)

		types.CalcSize(tk)
		types.CalcSize(te)

		// make and initialize static arrays
		vstatk := readonlystaticname(tk)
		vstate := readonlystaticname(te)

		datak := ir.NewCompLitExpr(base.Pos, ir.OARRAYLIT, nil, nil)
		datae := ir.NewCompLitExpr(base.Pos, ir.OARRAYLIT, nil, nil)
		for _, r := range entries {
			r := r.(*ir.KeyExpr)
			datak.List.Append(r.Key)
			datae.List.Append(r.Value)
		}
		fixedlit(inInitFunction, initKindStatic, datak, vstatk, init)
		fixedlit(inInitFunction, initKindStatic, datae, vstate, init)

		// loop adding structure elements to map
		// for i = 0; i < len(vstatk); i++ {
		//	map[vstatk[i]] = vstate[i]
		// }
		i := typecheck.Temp(types.Types[types.TINT])
		rhs := ir.NewIndexExpr(base.Pos, vstate, i)
		rhs.SetBounded(true)

		kidx := ir.NewIndexExpr(base.Pos, vstatk, i)
		kidx.SetBounded(true)
		lhs := ir.NewIndexExpr(base.Pos, m, kidx)

		zero := ir.NewAssignStmt(base.Pos, i, ir.NewInt(0))
		cond := ir.NewBinaryExpr(base.Pos, ir.OLT, i, ir.NewInt(tk.NumElem()))
		incr := ir.NewAssignStmt(base.Pos, i, ir.NewBinaryExpr(base.Pos, ir.OADD, i, ir.NewInt(1)))

		var body ir.Node = ir.NewAssignStmt(base.Pos, lhs, rhs)
		body = typecheck.Stmt(body) // typechecker rewrites OINDEX to OINDEXMAP
		body = orderStmtInPlace(body, map[string][]*ir.Name{})

		loop := ir.NewForStmt(base.Pos, nil, cond, incr, nil)
		loop.Body = []ir.Node{body}
		*loop.PtrInit() = []ir.Node{zero}

		appendWalkStmt(init, loop)
		return
	}
	// For a small number of entries, just add them directly.

	// Build list of var[c] = expr.
	// Use temporaries so that mapassign1 can have addressable key, elem.
	// TODO(josharian): avoid map key temporaries for mapfast_* assignments with literal keys.
	tmpkey := typecheck.Temp(m.Type().Key())
	tmpelem := typecheck.Temp(m.Type().Elem())

	for _, r := range entries {
		r := r.(*ir.KeyExpr)
		index, elem := r.Key, r.Value

		ir.SetPos(index)
		appendWalkStmt(init, ir.NewAssignStmt(base.Pos, tmpkey, index))

		ir.SetPos(elem)
		appendWalkStmt(init, ir.NewAssignStmt(base.Pos, tmpelem, elem))

		ir.SetPos(tmpelem)
		var a ir.Node = ir.NewAssignStmt(base.Pos, ir.NewIndexExpr(base.Pos, m, tmpkey), tmpelem)
		a = typecheck.Stmt(a) // typechecker rewrites OINDEX to OINDEXMAP
		a = orderStmtInPlace(a, map[string][]*ir.Name{})
		appendWalkStmt(init, a)
	}

	appendWalkStmt(init, ir.NewUnaryExpr(base.Pos, ir.OVARKILL, tmpkey))
	appendWalkStmt(init, ir.NewUnaryExpr(base.Pos, ir.OVARKILL, tmpelem))
}
```

如果字面量的个数大于 25 ，则会构件两个数组专门存储 key 和 value，在运行时循环添加数据。如果字面量的个数小于 25，则会在运行时初始化中直接添加的方式进行赋值。



#### map 访问原理

对 map 的访问有两种形式，一种是返回单值 `v := m[key]`，另一种是返回多值 `v, ok := m[key]`。由于 Go 语言没有函数重载的概念，所以这是在编译时决定返回一个值还是两个值。最终返回单个值的访问会被编译成 `mapaccess1_XXX` 函数，而返回多个值的访问会被编译成 `mapaccess2_XXX` 函数。

```go
//
// runtime/map.go
//

// mapaccess1 returns a pointer to h[key].  Never returns nil, instead
// it will return a reference to the zero object for the elem type if
// the key is not in the map.
// NOTE: The returned pointer may keep the whole map live, so don't
// hold onto it for very long.
func mapaccess1(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
	if raceenabled && h != nil {
		callerpc := getcallerpc()
		pc := funcPC(mapaccess1)
		racereadpc(unsafe.Pointer(h), callerpc, pc)
		raceReadObjectPC(t.key, key, callerpc, pc)
	}
	if msanenabled && h != nil {
		msanread(key, t.key.size)
	}
	if h == nil || h.count == 0 {
		if t.hashMightPanic() {
			t.hasher(key, 0) // see issue 23734
		}
		return unsafe.Pointer(&zeroVal[0])
	}
	if h.flags&hashWriting != 0 {
		throw("concurrent map read and map write")
	}
	hash := t.hasher(key, uintptr(h.hash0))
	m := bucketMask(h.B)
	b := (*bmap)(add(h.buckets, (hash&m)*uintptr(t.bucketsize)))
	if c := h.oldbuckets; c != nil {
		if !h.sameSizeGrow() {
			// There used to be half as many buckets; mask down one more power of two.
			m >>= 1
		}
		oldb := (*bmap)(add(c, (hash&m)*uintptr(t.bucketsize)))
		if !evacuated(oldb) {
			b = oldb
		}
	}
	top := tophash(hash)
bucketloop:
	for ; b != nil; b = b.overflow(t) {
		for i := uintptr(0); i < bucketCnt; i++ {
			if b.tophash[i] != top {
				if b.tophash[i] == emptyRest {
					break bucketloop
				}
				continue
			}
			k := add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))
			if t.indirectkey() {
				k = *((*unsafe.Pointer)(k))
			}
			if t.key.equal(key, k) {
				e := add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.elemsize))
				if t.indirectelem() {
					e = *((*unsafe.Pointer)(e))
				}
				return e
			}
		}
	}
	return unsafe.Pointer(&zeroVal[0])
}
```

Go 语言采用了一种简单的方式 `hash&m` 计算指定的 key 应该位于哪一个桶中，在获取桶的索引之后，`tophash(hash)` 计算出 hash 的前 8 位。接着就会与桶中的 tophash 进行对比。如果哈希值相同，则会找到相对应的 key 并检查是否相同。



#### map 赋值操作原理

和 map 访问类似，赋值操作最终会调用 `mapassign`  函数。在赋值时 map 必须已经进行了初始化，否则在运行时会报错为 assignment to entry in nil map。同时在执行写入时，会检查 flags 标志位，如果正在执行写入操作，则当前写入操作会报错。

```go
//
// runtime/map.go
//

// Like mapaccess, but allocates a slot for the key if it is not present in the map.
func mapassign(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
	if h == nil {
		panic(plainError("assignment to entry in nil map"))
	}
	if raceenabled {
		callerpc := getcallerpc()
		pc := funcPC(mapassign)
		racewritepc(unsafe.Pointer(h), callerpc, pc)
		raceReadObjectPC(t.key, key, callerpc, pc)
	}
	if msanenabled {
		msanread(key, t.key.size)
	}
	if h.flags&hashWriting != 0 {
		throw("concurrent map writes")
	}
	hash := t.hasher(key, uintptr(h.hash0))

	// Set hashWriting after calling t.hasher, since t.hasher may panic,
	// in which case we have not actually done a write.
	h.flags ^= hashWriting

	if h.buckets == nil {
		h.buckets = newobject(t.bucket) // newarray(t.bucket, 1)
	}

again:
	bucket := hash & bucketMask(h.B)
	if h.growing() {
		growWork(t, h, bucket)
	}
	b := (*bmap)(add(h.buckets, bucket*uintptr(t.bucketsize)))
	top := tophash(hash)

	var inserti *uint8
	var insertk unsafe.Pointer
	var elem unsafe.Pointer
bucketloop:
	for {
		for i := uintptr(0); i < bucketCnt; i++ {
			if b.tophash[i] != top {
				if isEmpty(b.tophash[i]) && inserti == nil {
					inserti = &b.tophash[i]
					insertk = add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))
					elem = add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.elemsize))
				}
				if b.tophash[i] == emptyRest {
					break bucketloop
				}
				continue
			}
			k := add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))
			if t.indirectkey() {
				k = *((*unsafe.Pointer)(k))
			}
			if !t.key.equal(key, k) {
				continue
			}
			// already have a mapping for key. Update it.
			if t.needkeyupdate() {
				typedmemmove(t.key, k, key)
			}
			elem = add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.elemsize))
			goto done
		}
		ovf := b.overflow(t)
		if ovf == nil {
			break
		}
		b = ovf
	}

	// Did not find mapping for key. Allocate new cell & add entry.

	// If we hit the max load factor or we have too many overflow buckets,
	// and we're not already in the middle of growing, start growing.
	if !h.growing() && (overLoadFactor(h.count+1, h.B) || tooManyOverflowBuckets(h.noverflow, h.B)) {
		hashGrow(t, h)
		goto again // Growing the table invalidates everything, so try again
	}

	if inserti == nil {
		// The current bucket and all the overflow buckets connected to it are full, allocate a new one.
		newb := h.newoverflow(t, b)
		inserti = &newb.tophash[0]
		insertk = add(unsafe.Pointer(newb), dataOffset)
		elem = add(insertk, bucketCnt*uintptr(t.keysize))
	}

	// store new key/elem at insert position
	if t.indirectkey() {
		kmem := newobject(t.key)
		*(*unsafe.Pointer)(insertk) = kmem
		insertk = kmem
	}
	if t.indirectelem() {
		vmem := newobject(t.elem)
		*(*unsafe.Pointer)(elem) = vmem
	}
	typedmemmove(t.key, insertk, key)
	*inserti = top
	h.count++

done:
	if h.flags&hashWriting == 0 {
		throw("concurrent map writes")
	}
	h.flags &^= hashWriting
	if t.indirectelem() {
		elem = *((*unsafe.Pointer)(elem))
	}
	return elem
}
```

如果发现当前 map 正在重建，则会优先完成重建过程。最后会计算 tophash，如果能找到相同的 tophash，则直接更新 value。如果找不到还需要在溢出桶中进行查找。如果桶中没有空闲位置时，则会申请一个新的桶（该桶会先从 extra 字段中获取）。



#### map 的重建原理

重建时会调用 hashGrow 函数，如果因为负载因子超载，则会进行双倍重建。当溢出桶的数量太多时，会进行等量重建。

```go
//
// runtime/map.go
//

func hashGrow(t *maptype, h *hmap) {
	// If we've hit the load factor, get bigger.
	// Otherwise, there are too many overflow buckets,
	// so keep the same number of buckets and "grow" laterally.
	bigger := uint8(1)
	if !overLoadFactor(h.count+1, h.B) {
		bigger = 0
		h.flags |= sameSizeGrow
	}
	oldbuckets := h.buckets
	newbuckets, nextOverflow := makeBucketArray(t, h.B+bigger, nil)

	flags := h.flags &^ (iterator | oldIterator)
	if h.flags&iterator != 0 {
		flags |= oldIterator
	}
	// commit the grow (atomic wrt gc)
	h.B += bigger
	h.flags = flags
	h.oldbuckets = oldbuckets
	h.buckets = newbuckets
	h.nevacuate = 0
	h.noverflow = 0

	if h.extra != nil && h.extra.overflow != nil {
		// Promote current overflow buckets to the old generation.
		if h.extra.oldoverflow != nil {
			throw("oldoverflow is not nil")
		}
		h.extra.oldoverflow = h.extra.overflow
		h.extra.overflow = nil
	}
	if nextOverflow != nil {
		if h.extra == nil {
			h.extra = new(mapextra)
		}
		h.extra.nextOverflow = nextOverflow
	}

	// the actual copying of the hash table data is done incrementally
	// by growWork() and evacuate().
}
```

数据转移遵循写时复制原则，只有在正在赋值时才会选择是否进行数据转移。



#### map 删除原理

删除操作同样需要根据 key 计算出桶的索引和 tophash，并在需要时查找溢出桶。如果找到则会清空该数据，并根据情况置位为 emptyOne 或者 emptyRest。

```go
//
// runtime/map.go
//

func mapdelete(t *maptype, h *hmap, key unsafe.Pointer) {
	if raceenabled && h != nil {
		callerpc := getcallerpc()
		pc := funcPC(mapdelete)
		racewritepc(unsafe.Pointer(h), callerpc, pc)
		raceReadObjectPC(t.key, key, callerpc, pc)
	}
	if msanenabled && h != nil {
		msanread(key, t.key.size)
	}
	if h == nil || h.count == 0 {
		if t.hashMightPanic() {
			t.hasher(key, 0) // see issue 23734
		}
		return
	}
	if h.flags&hashWriting != 0 {
		throw("concurrent map writes")
	}

	hash := t.hasher(key, uintptr(h.hash0))

	// Set hashWriting after calling t.hasher, since t.hasher may panic,
	// in which case we have not actually done a write (delete).
	h.flags ^= hashWriting

	bucket := hash & bucketMask(h.B)
	if h.growing() {
		growWork(t, h, bucket)
	}
	b := (*bmap)(add(h.buckets, bucket*uintptr(t.bucketsize)))
	bOrig := b
	top := tophash(hash)
search:
	for ; b != nil; b = b.overflow(t) {
		for i := uintptr(0); i < bucketCnt; i++ {
			if b.tophash[i] != top {
				if b.tophash[i] == emptyRest {
					break search
				}
				continue
			}
			k := add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))
			k2 := k
			if t.indirectkey() {
				k2 = *((*unsafe.Pointer)(k2))
			}
			if !t.key.equal(key, k2) {
				continue
			}
			// Only clear key if there are pointers in it.
			if t.indirectkey() {
				*(*unsafe.Pointer)(k) = nil
			} else if t.key.ptrdata != 0 {
				memclrHasPointers(k, t.key.size)
			}
			e := add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.elemsize))
			if t.indirectelem() {
				*(*unsafe.Pointer)(e) = nil
			} else if t.elem.ptrdata != 0 {
				memclrHasPointers(e, t.elem.size)
			} else {
				memclrNoHeapPointers(e, t.elem.size)
			}
			b.tophash[i] = emptyOne
			// If the bucket now ends in a bunch of emptyOne states,
			// change those to emptyRest states.
			// It would be nice to make this a separate function, but
			// for loops are not currently inlineable.
			if i == bucketCnt-1 {
				if b.overflow(t) != nil && b.overflow(t).tophash[0] != emptyRest {
					goto notLast
				}
			} else {
				if b.tophash[i+1] != emptyRest {
					goto notLast
				}
			}
			for {
				b.tophash[i] = emptyRest
				if i == 0 {
					if b == bOrig {
						break // beginning of initial bucket, we're done.
					}
					// Find previous bucket, continue at its last entry.
					c := b
					for b = bOrig; b.overflow(t) != c; b = b.overflow(t) {
					}
					i = bucketCnt - 1
				} else {
					i--
				}
				if b.tophash[i] != emptyOne {
					break
				}
			}
		notLast:
			h.count--
			// Reset the hash seed to make it more difficult for attackers to
			// repeatedly trigger hash collisions. See issue 25237.
			if h.count == 0 {
				h.hash0 = fastrand()
			}
			break search
		}
	}

	if h.flags&hashWriting == 0 {
		throw("concurrent map writes")
	}
	h.flags &^= hashWriting
}
```

