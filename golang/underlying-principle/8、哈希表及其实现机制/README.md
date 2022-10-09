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

