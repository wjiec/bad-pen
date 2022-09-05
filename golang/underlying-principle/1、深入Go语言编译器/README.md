一、深入Go语言编译器
---------------------------------

Go 语言编译器不仅能准确地翻译高级语言，也能进行代码优化。

编译器是一个大型且复杂的系统，一个号的编译器会很好地结合形式语言理论、算法、人工智能、系统设计、计算机体系结构及编程语言理论。Go语言的编译器遵循主流编译器采用的经典策略及相似的处理流程和优化规则（例如经典的递归下降的语法解析、抽象语法树的构建）。

> 和 Go 语言编译器有关的代码主要位于 src/cmd/compile/internal 目录下



### Go语言编译器的阶段

在经典的编译原理中，一般将编译器分为编译前段、优化器和编译器后端。这种编译器被称为三阶段编译器。

* **编译器前端**：专注于理解源程序、扫描解析源程序并进行精准的语义表达。
* **中间阶段（Intermediate Representation，IR）**：编译器会使用多个 IR 阶段、多种数据结构表示代码，并在中间阶段对代码进行多次优化。
* **编译器后端**：专注于生产特定目标机器上的程序，这种程序可能是可自行文件、也可能是需要进一步处理的中间 obj 文件、汇编语言等。

Go 语言编译器的执行流程可细分为多个阶段，主要有词法分析、语法解析、抽象语法树构建、类型检查、变量捕获、函数内联、逃逸分析、闭包重写、遍历函数、SSA 生成、机器码生成阶段。



#### 词法分析

词法分析阶段，Go 语言编译器会扫描输入的 Go 源文件，并将其符号（Token）化。这些 Token 实质上是用 iota 声明的整数，定义在 syntax/tokens.go 中。符号化保留了 Go 语言中定义的符号，可以识别出错误的拼写。

我们可以使用 Go 标准库 `go/scanner` 和 `go/token` 提供的接口用于扫描源代码：

```go
func ScannerErrorHandler(pos token.Position, msg string) {
	_, _ = fmt.Fprintf(os.Stderr, "ERROR: %s(%s)", msg, pos)
}

func main() {
	src := []byte("v := cos(x) + sin(x) * 2i // Euler")

	fSet := token.NewFileSet()
	f := fSet.AddFile("", fSet.Base(), len(src))

	var s scanner.Scanner
	s.Init(f, src, ScannerErrorHandler, scanner.ScanComments)

	fmt.Printf("%-8s\t%-12s\t%s\n", "<POSITION>", "<TOKEN>", "<LITERAL>")
	for {
		// position, token, literal
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}

		fmt.Printf("%-8s\t%-12s\t%q\n", fSet.Position(pos), tok, lit)
	}
}

// <POSITION>	<TOKEN>     	<LITERAL>
// 1:1     	IDENT       	"v"
// 1:3     	:=          	""
// 1:6     	IDENT       	"cos"
// 1:9     	(           	""
// 1:10    	IDENT       	"x"
// 1:11    	)           	""
// 1:13    	+           	""
// 1:15    	IDENT       	"sin"
// 1:18    	(           	""
// 1:19    	IDENT       	"x"
// 1:20    	)           	""
// 1:22    	*           	""
// 1:24    	IMAG        	"2i"
// 1:27    	;           	"\n"
// 1:27    	COMMENT     	"// Euler"
```



#### 语法解析

词法分析结束后，需要根据 Go语言中指定的语法对符号化后的 Go 文件进行解析。Go 语言采用标准的自上而下的递归下降（Top-Down Recursive-Descent）算法，以简单高效的方式完成无须回溯的语法分析，核心算法位于 syntax/nodes.go 以及 syntax/parser.go 中。

语法解析后会将语义存储到对应的结构体中，结构体所对应的层次结构是构建抽象语法树的基础。



#### 抽象语法树构建

编译器前段必须构建程序的中间表示形式，以便在编译器中间阶段及后端使用。抽象语法树（Abstract Syntax Tree）是一种常见的树状结构的中间态。Go 语言中的任何一种 import、type、const、func 声明都是一个根节点，在根节点下包含当前声明的子节点。核心逻辑位于 syntax/noder.go 文件中。



#### 类型检查

完成抽象语法树的初步构建滞后，就进入类型检查阶段，编译器会遍历节点树并决定节点的类型。在类型检查阶段，会对一些类型做特别的语法或语义检查（如引用的结构体字段是否是大写可导出的，数组字面量的访问是否超过其长度，数组的索引是不是正整数等）。除此之外，在类型检查阶段还会进行其他工作，例如计算编译时常量、将标识符与声明绑定等。类型检查的核心逻辑位于 syntax/typecheck.go 中。



#### 变量捕获

在类型检查阶段结束后，Go 编译器将对抽象语法树进行分析及重构，从而完成一系列优化。**变量捕获主要是针对闭包场景**而言的，由于闭包函数中可能引用闭包外的变量，因此变量捕获需要明确在闭包中是通过**值引用**还是地址引用的方式来捕获变量。

```go
package main

func main() {
	a := 1
	b := 2

	go func() {
		println(a, b)
	}()

	a = -1
}
```

我们可以使用 `go tool compile -m=2 main.go` 得到以下优化日志：

```plain
...
main.go:4:2: main capturing by ref: a (addr=false assign=true width=8)
...
main.go:5:2: main capturing by value: b (addr=false assign=false width=8)
...
```

可见，由于 a 在闭包之后还进行了其他赋值操作，所以必须采用地址引用方式进行捕获。这部分的核心逻辑位于 gc/closure.go 的 capturevars 函数中。



#### 函数内联

函数内联指将较小的函数直接组合进调用者的函数，这是现代编译器优化的一种核心技术。函数内联的优势在于，可以减少函数调用带来的开销（参数、返回值的栈复制、栈寄存器开销、函数前、后的栈扩容检查）。同时函数内联也是其他编译器优化的基础。

Go语言编译器会计算函数内联话费的成本，只有执行相对简单的函数时才会内联（如包含 for、range、go、select 等语句时，或函数包含太多的语句、或是递归函数时不会执行内联）。

```go
package main

func simple(s string) {
	println("hello", s)
}

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func main() {
	simple("golang")
	fib(10)
}
```

我们通过 `go tool compile -m=2 main.go` 可以得到以下优化日志：

```plain
main.go:3:6: can inline simple with cost 3 as: func(string) { println("hello", s) }
main.go:7:6: cannot inline fib: recursive
...
```

函数内联优化的核心逻辑位于 gc/inl.go 中



#### 逃逸分析

逃逸分析是 Go 语言中重要的优化阶段，用于标识变量内存应该被分配到栈区还是堆区。Go 语言能够通过编译时的逃逸分析识别到函数是否返回了栈上指针等问题，自动将对应变量放置到堆区，并借助 Go 运行时的垃圾回收机制自动释放内存。编译器在优化过程中会尽量将变量放置到栈中，因为栈中的对象会随着函数调用结束而自动销毁，减轻运行时分配和垃圾回收的负担。

在 Go 语言中，开发者模糊了栈区和堆区的区别，任何对象都有可能被分配到栈中，也有可能被分配到堆中。分配对象时，遵循以下两个原则：

* 原则一：指向栈上对象的指针不能被存储到堆区
* 原则二：指向栈上对象的指针不能超过栈对象的生命周期

Go 语言通过对抽象语法树的静态数据流（static data-flow analysis）来实现逃逸分析，通过构建带权重的有向图来查找负权重的方式实现。核心逻辑位于 gc/escape.go 中。



#### 闭包重写

在完成逃逸分析之后，闭包重写阶段的主要工作是将闭包调用分为**闭包后被立即调用**和**闭包定义后不被立即调用**两种情况。在闭包被立即调用情况下，闭包只能被调用一次，这时可以将闭包转换为普通函数的调用形式。如果闭包不被立即调用，而是后续调用，那么同一个闭包可能被调用多次，这时就需要创建闭包对象。闭包重写的核心逻辑位于 gc/closure.go 中



#### 遍历函数

闭包重写之后的遍历函数阶段的核心逻辑位于 gc/walk.go 文件的 walk 函数中。该阶段会识别出声明但未被使用的变量，遍历函数中的声明和表达式，将某些代表操作的节点转换为运行时的具体函数执行。例如，对于 new 操作，如果变量发生了逃逸，那么最终会调用运行时 newobject 函数将对象分配到堆区。



#### SSA生成

