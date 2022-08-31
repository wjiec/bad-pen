一、深入Go语言编译器
---------------------------------

Go 语言编译器不仅能准确地翻译高级语言，也能进行代码优化。

编译器是一个大型且复杂的系统，一个号的编译器会很好地结合形式语言理论、算法、人工智能、系统设计、计算机体系结构及编程语言理论。Go语言的编译器遵循主流编译器采用的经典策略及相似的处理流程和优化规则（例如经典的递归下降的语法解析、抽象语法树的构建）。

> 和 Go 语言编译器有关的代码主要位于 src/cmd/compile/internal 目录下



### Go语言编译器的阶段

在经典的编译原理中，一般将编译器分为编译前段、优化器和编译器后端。这种编译器被称为三阶段编译器。

* 编译器前端：专注于理解源程序、扫描解析源程序并进行精准的语义表达。
* 中间阶段（Intermediate Representation，IR）：编译器会使用多个 IR 阶段、多种数据结构表示代码，并在中间阶段对代码进行多次优化。
* 编译器后端：专注于生产特定目标机器上的程序，这种程序可能是可自行文件、也可能是需要进一步处理的中间 obj 文件、汇编语言等。

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

