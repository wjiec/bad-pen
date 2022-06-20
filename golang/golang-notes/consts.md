常量
------

1、在常量组中如不指定类型和初始值，则与上一行非空常量右值相同

```go
const (
	A int = 120
	B
	C string = "hello, world"
	D
)

func main() {
	fmt.Printf("var B %T = %d\n", B, B)
	fmt.Printf("var D %T = %q\n", D, D)
	// var B int = 120
	// var D string = "hello, world"
}
```



iota
-----

1、可以在常量定义中使用多个 iota ，他们将各自独立计数，只需要保证组内每行常量的数量相同即可

```go
const (
	A, _ = iota, 1 << (iota * 10)
	B, K
	C, M
	D, G
)

func main() {
	fmt.Printf("A = %d, _ = 0\n", A)
	fmt.Printf("B = %d, K = %d\n", B, K)
	fmt.Printf("C = %d, M = %d\n", C, M)
	fmt.Printf("D = %d, G = %d\n", D, G)
	// A = 0, _ = 0
	// B = 1, K = 1024
	// C = 2, M = 1048576
	// D = 3, G = 1073741824
}
```

2、如果中断 iota 自增，则后续自增值按行序递增

```go
const (
	A = iota // 0
	B        // 1
	C = 100  // 100
	D        // 100
	E = iota // 4
	F        // 5

	X = iota // 6
	Y        // 7
	Z        // 8

	O = iota + 1 // 10 = 9 + 1
	P            // 11
	Q            // 12
)

func main() {
	fmt.Printf("A = %d\n", A)
	fmt.Printf("B = %d\n", B)
	fmt.Printf("C = %d\n", C)
	fmt.Printf("D = %d\n", D)
	fmt.Printf("E = %d\n", E)
	fmt.Printf("F = %d\n", F)
	// A = 0
	// B = 1
	// C = 100
	// D = 100
	// E = 4
	// F = 5

	fmt.Printf("X = %d\n", X)
	fmt.Printf("Y = %d\n", Y)
	fmt.Printf("Z = %d\n", Z)
	// X = 6
	// Y = 7
	// Z = 8

	fmt.Printf("O = %d\n", O)
	fmt.Printf("P = %d\n", P)
	fmt.Printf("Q = %d\n", Q)
	// O = 10
	// P = 11
	// Q = 12
}
```

