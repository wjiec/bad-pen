函数
------

1、函数中的可变参数是通过切片的方式实现的，所以参数在传递过程中并不会拷贝底层数组

```go
func Fill(v int, a ...int) {
	for i := 0; i < len(a); i++ {
		a[i] = v
	}
}

func main() {
	s := []int{1, 2, 3, 4, 5}
	Fill(-1, s...)

	fmt.Printf("s = %v\n", s)
	// s = [-1 -1 -1 -1 -1]

	a := [...]int{1, 2, 3, 4, 5}
	Fill(-1, a[:]...)

	fmt.Printf("a = %v\n", s)
	// a = [-1 -1 -1 -1 -1]
}
```

2、defer 只会保留最后一次的方法调用，其余的都会在之前执行

```go
var after = false

type Printer func(s string) Printer

func Print(s string) Printer {
	fmt.Printf("s = %s, after = %v\n", s, after)
	return Print
}

func main() {
	defer Print("a")("b")("c")("d")

	after = true
	// s = a, after = false
	// s = b, after = false
	// s = c, after = false
	// s = d, after = true
}
```



### 注意事项

