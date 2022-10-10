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

