变量及数据
----------------



1、字符串操作通常在堆上分配内存，这会对 Web 等高并发应用造成较大影响，因为在字符串操作过程中会有大量的字符串对象要做垃圾回收。**建议使用 `[]byte` 缓存池，或在栈上自行拼装等方式来实现 zero-garbage**

```go
//
// strings.go
//

func Join(ss ...string) string {
	return strings.Join(ss, "")
}

func ConcatMulti(ss ...string) string {
	s := ss[15]
	s += ss[14]
	s += ss[13]
	s += ss[12]
	s += ss[11]
	s += ss[10]
	s += ss[9]
	s += ss[8]
	s += ss[7]
	s += ss[6]
	s += ss[5]
	s += ss[4]
	s += ss[3]
	s += ss[2]
	s += ss[1]
	s += ss[0]

	return s
}

func ConcatSingle(ss ...string) string {
	return ss[15] + ss[14] + ss[13] + ss[12] + ss[11] + ss[10] + ss[9] + ss[8] +
		ss[7] + ss[6] + ss[5] + ss[4] + ss[3] + ss[2] + ss[1] + ss[0]
}

//
// strings_test.go
//

func BenchmarkJoin(b *testing.B) {
	s := make([]string, 16)
	for i := 0; i < len(s); i++ {
		s[i] = "a"
		if i != 0 {
			s[i] += s[i-1]
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Join(s...)
	}

	// BenchmarkJoin
	// BenchmarkJoin-8           	 4379310	       252.1 ns/op
}

func BenchmarkConcatMulti(b *testing.B) {
	s := make([]string, 16)
	for i := 0; i < len(s); i++ {
		s[i] = "a"
		if i != 0 {
			s[i] += s[i-1]
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcatMulti(s...)
	}

	// BenchmarkConcatMulti
	// BenchmarkConcatMulti-8    	 1000000	      1438 ns/op
}

func BenchmarkConcatSingle(b *testing.B) {
	s := make([]string, 16)
	for i := 0; i < len(s); i++ {
		s[i] = "a"
		if i != 0 {
			s[i] += s[i-1]
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcatSingle(s...)
	}

	// BenchmarkConcatSingle
	// BenchmarkConcatSingle-8   	 5454232	       241.8 ns/op
}
```



2、因内存访问安全和哈希算法等缘故，字典被设计成“not addressable”的，故不能直接修改value成员。或者可以使用指针类型

```go
type User struct {
	Name string
	Age  int
}

func main() {
	m1 := make(map[string]User)
	m1["hello"] = User{Name: "hello", Age: 1}

	//m1["hello"].Name = "world" // cannot assign to struct field m1["hello"].Name in map

	m2 := make(map[string]*User)
	m2["world"] = &User{Name: "world", Age: 2}

	m2["world"].Name = "foobar"
	fmt.Println(m2)
	// map[world:0xc000004078]
}
```



3、不能对 nil 字典进行写操作，但是可以读（得到零值）

```go
type User struct {
	Name string
	Age  int
}

func main() {
	var m map[string]User // m == nil

	fmt.Printf("nil map => %#v", m["hello"])
	// nil map => main.User{Name:"", Age:0}

	//m["world"] = User{} // panic: assignment to entry in nil map
}
```





4、在迭代期间删除或新增键值是安全的

```go
func main() {
	m := make(map[int]int)
	for i := 0; i < 10; i++ {
		m[i] = rand.Intn(128)
	}

	for k, v := range m {
		fmt.Printf("k = %d, v = %d, m = %#v\n", k, v, m)
		if v < k {
			delete(m, k)
		}
		m[v] = rand.Intn(128)
		// k = 5, v = 6, m = map[int]int{0:33, 1:15, 2:71, 3:59, 4:1, 5:6, 6:57, 7:44, 8:72, 9:36}
		// k = 7, v = 44, m = map[int]int{0:33, 1:15, 2:71, 3:59, 4:1, 5:6, 6:70, 7:44, 8:72, 9:36}
		// k = 3, v = 59, m = map[int]int{0:33, 1:15, 2:71, 3:59, 4:1, 5:6, 6:70, 7:44, 8:72, 9:36, 44:47}
		// k = 1, v = 15, m = map[int]int{0:33, 1:15, 2:71, 3:59, 4:1, 5:6, 6:70, 7:44, 8:72, 9:36, 44:47, 59:34}
		// ...
	}
}
```



5、空结构（`struct{}`）是指没有字段的结构类型，它比较特殊，因为无论是其本身还是作为数组元素类型，其长度都为零`

```go
func main() {
	var es struct{}
	var ea [100]struct{}

	fmt.Printf("sizeof(es) = %d\n", unsafe.Sizeof(es))
	fmt.Printf("sizeof(ea) = %d\n", unsafe.Sizeof(ea))
	// sizeof(es) = 0
	// sizeof(ea) = 0

	fmt.Printf("&es = %p\n", &es)
	fmt.Printf("&ea = %p\n", &ea)
	// &es = 0x5972f8
	// &ea = 0x5972f8

	// see runtime.zerobase
}
```

实际上，所有长度为 0 的对象通常都指向 `runtime.zerobase` 变量



6、在分配内存时，字段需做对齐处理，通常是以所有字段中最长的基础类型宽度为标准

```go
type t1 struct {
	b1  byte  // size = 1, offset = 0
	i16 int16 // size = 2, offset = 2
	i32 int32 // size = 4, offset = 4
}

type t2 struct {
	b1 byte // size = 1, offset = 0
	b2 byte // size = 1, offset = 1
}

type t3 struct {
	b1  byte  // size = 1, offset = 0
	i16 int16 // size = 2, offset = 2
	i32 int32 // size = 4, offset = 4
	i64 int64 // size = 8, offset = 8
}

func main() {
	fmt.Printf("t1, align = %d, size = %d\n", unsafe.Alignof(t1{}), unsafe.Sizeof(t1{}))
	fmt.Printf("t2, align = %d, size = %d\n", unsafe.Alignof(t2{}), unsafe.Sizeof(t2{}))
	fmt.Printf("t3, align = %d, size = %d\n", unsafe.Alignof(t3{}), unsafe.Sizeof(t3{}))
	// t1, align = 4, size = 8
	// t2, align = 1, size = 2
	// t3, align = 8, size = 16
}
```

