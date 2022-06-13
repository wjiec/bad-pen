语言陷阱
-------------

Go 语言的语法简单，类型系统设计“短小精悍”，但也不是完美无瑕。



### 多值赋值和短变量声明

Go 语言支持多值赋值，在函数或方法内部也支持短变量声明并赋值。



#### 多值赋值

多值赋值包括两层语义：

1、对左侧操作数中的表达式、索引值进行计算和确定，首先确定左侧操作数的地址；然后对右侧的赋值表达式进行计算，如果发现右侧的表达式计算引用了左侧的变量，则**创建临时变量进行值拷贝**，最后完成计算

2、从左到右的顺序依次赋值

```go
func main() {
	i := 0
	s := []int{1, 2, 3}

	// 首先直接计算左边的: 分别为 i, s[0]
	// 然后计算右边的: 分别为 s[0], 2
	// 综合起来就是 i, s[0] = s[0], 2
	i, s[i] = s[i], 2
	fmt.Printf("i = %d, s = %+v\n", i, s)
	// i = 1, s = [2 2 3]
}
```

我们可以通过反汇编的方式来验证是否有临时变量产生，首先我们有以下 Go 代码：

```go
func main() {
	a, b := 11, 22
	a, b = b, a
	fmt.Printf("a = %d, b = %d\n", a, b)
	// a = 22, b = 11
}
```

接着我们使用 `go tool compile -N -l -S` 进行反汇编获得汇编代码

```assembly
"".main STEXT size=307 args=0x0 locals=0x88 funcid=0x0
	0x0000 00000 (main.go:5)	TEXT	"".main(SB), ABIInternal, $136-0
	// 在需要的时候进行栈扩展
	0x0000 00000 (main.go:5)	LEAQ	-8(SP), R12
	0x0005 00005 (main.go:5)	CMPQ	R12, 16(R14)
	0x0009 00009 (main.go:5)	JLS	297

    // 开辟栈空间
	0x000f 00015 (main.go:5)	SUBQ	$136, SP
	0x0016 00022 (main.go:5)	MOVQ	BP, 128(SP)
	0x001e 00030 (main.go:5)	LEAQ	128(SP), BP
	
	// 初始化及多值赋值操作
	0x0026 00038 (main.go:6)	MOVQ	$11, "".a+32(SP) 			// a = 11
	0x002f 00047 (main.go:6)	MOVQ	$22, "".b+24(SP)			// b = 22
	0x0038 00056 (main.go:7)	MOVQ	"".a+32(SP), CX				// CX = a
	0x003d 00061 (main.go:7)	MOVQ	CX, ""..autotmp_3+40(SP)	// autotmp_3 = CX = a
	0x0042 00066 (main.go:7)	MOVQ	"".b+24(SP), CX				// CX = b
	0x0047 00071 (main.go:7)	MOVQ	CX, "".a+32(SP)				// (a = CX = b)
	0x004c 00076 (main.go:7)	MOVQ	""..autotmp_3+40(SP), CX	// CX = autotmp_3 = a
	0x0051 00081 (main.go:7)	MOVQ	CX, "".b+24(SP)				// (b = CX = a)
```

可以看到实际上使用了一次临时变量 `autotmp_3`



#### 短变量的声明和赋值

短变量声明和赋值的语法约定：

1、使用 `:=` 操作符，变量的定义和初始化同时完成

2、变量名后不要跟任何类型，Go 编译器可以靠右边的值进行推导

3、支持多值短变量声明和赋值

4、只能在函数和类型方法内部

使用短变量声明和赋值其中左侧必须要有一个是新定义的局部变量，对于已存在的变量将会执行赋值操作，而新变量则执行声明并赋值操作。如果声明的局部变量与全局变量同名，则会创建新的局部变量并屏蔽全局同名变量

```go
var (
	a = 11
	b = 22
)

func main() {
	fmt.Printf("a = %d, b = %d\n", a, b)
	// a = 11, b = 22

	a, b := 33, 44
	fmt.Printf("a = %d, b = %d\n", a, b)
	// a = 33, b = 44
}
```



### range 复用临时变量

在使用 `range` 对数组、切片、字典等进行遍历时，需要注意：**迭代变量是复用的**

```go
func main() {
	var wg sync.WaitGroup
	s := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	for _, v := range s {
		wg.Add(1)
		go func() {
			fmt.Println(v)
			// 6
			// 0
			// 5
			// 0
			// 0
			// 0
			// 0
			// 0
			// 0
			// 0
			wg.Done()
		}()
	}

	wg.Wait()
}
```

正确的写法是使用函数参数做一次复制，而不是使用闭包变量：

```go
func main() {
	var wg sync.WaitGroup
	s := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	for _, v := range s {
		wg.Add(1)
		go func(v int) {
			fmt.Println(v)
			// 1
			// 0
			// 2
			// 5
			// 3
			// 4
			// 6
			// 9
			// 7
			// 8
			wg.Done()
		}(v)
	}

	wg.Wait()
}
```



### defer 陷阱

`defer` 主要有两个副作用：

1、对返回值的影响

2、对性能的影响

对返回值的影响可以见以下代码：

```go
func Defer1() (r int) {
	// 这里对闭包变量 r 进行修改
	defer func() {
		r++
	}()
	return 0
}

func Defer2() (r int) {
	v := 5
	// 在对 r 赋值之后
	// 再修改 v 的值已经不会影响 r 了
	defer func() {
		v = v + 5
	}()
	return v
}

func Defer3() (r int) {
	v := 5
	// 形参屏蔽了闭包变量 r
	defer func(r int) {
		r += 5
	}(v)
	return v
}

func main() {
	fmt.Printf("Defer1() = %d\n", Defer1())
	// Defer1() = 1
	fmt.Printf("Defer2() = %d\n", Defer2())
	// Defer2() = 5
	fmt.Printf("Defer3() = %d\n", Defer3())
	// Defer3() = 5
}
```

对于所有带 `defer` 的函数返回值整体上有三个步骤：

1、执行 `return` 语句：将 `return` 后面跟着的表达式的结果**复制**到返回值所在的栈地址（如果使用不带表达式的 `return` 则此步骤不做任何动作）

2、执行 `defer` 语句，多个 `defer` 语句将按照 `FILO` 顺序执行

3、执行 `RET` 指令，返回上一层调用

由此可见，在 `defer` 语句中只能通过直接引用的方式对返回值进行修改。最好直接在定义函数时使用不带返回值名的方式。
