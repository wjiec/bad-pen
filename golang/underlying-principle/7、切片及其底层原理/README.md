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

