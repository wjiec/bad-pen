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

遍历函数后，编译器会将抽象语法树转换为下一个重要的中间表示形态，称为 SSA（Static Single Assignment，静态单赋值）。SSA 被大多数现在的编译器用作编译器后端，负责生成更有效的机器码。在 SSA 生成阶段，每个变量在声明之前都需要被定义，并且每个变量只会赋值一次。

*在 SSA 中条件判断等多分支情况会稍微复杂一点，为了解决这个问题，SSA 生成阶段会引入额外的函数来处理这种情况。*

SSA 生成阶段是编译器进行后续优化的保证，比如常量传播（Constant Propagation）、无效代码消除、消除冗余、强度降低（Strength Reduction）等。大部分 SSA 相关的代码位于 ssa/ 文件夹下，将抽象语法树转换为 SSA 的逻辑位于 gc/ssa.go 文件中。



#### 机器码生成 —— 汇编器

在 SSA 阶段，编译器先执行与特定指令集无关的优化，再执行与特定指令集有关的优化，并最终生成与特定指令集有关的指令和寄存器分配方式。在 SSA 后，编译器将调用与特定指令集有关的汇编器（Assembler）生成 obj 文件，而 obj 文件将被作为链接器（Linker）的输入，生成二进制可执行文件。汇编与链接的核心逻辑位于 internal/obj 目录中。



#### 机器码生成 —— 链接

链接就是将编写的程序与外部程序组合在一起的过程。链接分为静态链接和动态链接，静态链接的特点是链接器会将程序中使用的所有库程序复制到最后的可可执行文件中，而动态链接只会在最后的可执行文件中存储动态链接库的位置，并在运行时调用。因此静态链接更快，并且可移植，它不需要运行它的系统上存在该库，但是它会占用更多的磁盘和内存空间。
Go 代码在默认情况下是使用静态链接的，但是在一些特殊情况下，如在使用了 CGO 时，则会使用操作系统的动态链接库。我们可以通过在 go build 编译时指定 buildmode 参数来选择链接形式。

我们可以在 go build 中使用 -x 参数打印详细的编译过程。



#### ELF 文件解析

ELF（Executabl and Linkable Format）是类 UNIX 操作系统下最常见的可执行且可链接的文件格式。除机器码外，在可执行文件中还可能包含调试信息、动态链接库信息、符号表信息等。我们可以通过 `readelf -h <exe>` 查看 ELF 文件的头信息：

```bash
$ readelf -h main
ELF Header:
  Magic:   7f 45 4c 46 02 01 01 00 00 00 00 00 00 00 00 00 
  Class:                             ELF64
  Data:                              2's complement, little endian
  Version:                           1 (current)
  OS/ABI:                            UNIX - System V
  ABI Version:                       0
  Type:                              EXEC (Executable file)
  Machine:                           Advanced Micro Devices X86-64
  Version:                           0x1
  Entry point address:               0x463690
  Start of program headers:          64 (bytes into file)
  Start of section headers:          456 (bytes into file)
  Flags:                             0x0
  Size of this header:               64 (bytes)
  Size of program headers:           56 (bytes)
  Number of program headers:         7
  Size of section headers:           64 (bytes)
  Number of section headers:         25
  Section header string table index: 3
```

也可以通过 readelf 工具查看 EL 文件中的 section 的信息：

```bash
$ radelf -S main
There are 25 section headers, starting at offset 0x1c8:

Section Headers:
  [Nr] Name              Type             Address           Offset
       Size              EntSize          Flags  Link  Info  Align
  [ 0]                   NULL             0000000000000000  00000000
       0000000000000000  0000000000000000           0     0     0
  [ 1] .text             PROGBITS         0000000000401000  00001000
       00000000002e7cc4  0000000000000000  AX       0     0     16
  [ 2] .rodata           PROGBITS         00000000006e9000  002e9000
       0000000000148920  0000000000000000   A       0     0     32
  [ 3] .shstrtab         STRTAB           0000000000000000  00431920
       00000000000001bc  0000000000000000           0     0     1
  [ 4] .typelink         PROGBITS         0000000000831ae0  00431ae0
       0000000000002368  0000000000000000   A       0     0     32
  [ 5] .itablink         PROGBITS         0000000000833e48  00433e48
       0000000000000b58  0000000000000000   A       0     0     8
  [ 6] .gosymtab         PROGBITS         00000000008349a0  004349a0
       0000000000000000  0000000000000000   A       0     0     1
  [ 7] .gopclntab        PROGBITS         00000000008349a0  004349a0
       00000000001db3a3  0000000000000000   A       0     0     32
  [ 8] .go.buildinfo     PROGBITS         0000000000a10000  00610000
       0000000000000020  0000000000000000  WA       0     0     16
  [ 9] .noptrdata        PROGBITS         0000000000a10020  00610020
       0000000000037840  0000000000000000  WA       0     0     32
  [10] .data             PROGBITS         0000000000a47860  00647860
       000000000000ad50  0000000000000000  WA       0     0     32
  [11] .bss              NOBITS           0000000000a525c0  006525c0
       000000000002c670  0000000000000000  WA       0     0     32
  [12] .noptrbss         NOBITS           0000000000a7ec40  0067ec40
       00000000000034e8  0000000000000000  WA       0     0     32
  [13] .zdebug_abbrev    PROGBITS         0000000000a83000  00653000
       0000000000000119  0000000000000000           0     0     8
  [14] .zdebug_line      PROGBITS         0000000000a83119  00653119
       000000000006b413  0000000000000000           0     0     8
  [15] .zdebug_frame     PROGBITS         0000000000aee52c  006be52c
       0000000000018b68  0000000000000000           0     0     8
  [16] .zdebug_pubnames  PROGBITS         0000000000b07094  006d7094
       0000000000003a0a  0000000000000000           0     0     8
  [17] .zdebug_pubtypes  PROGBITS         0000000000b0aa9e  006daa9e
       000000000000ba7c  0000000000000000           0     0     8
  [18] .debug_gdb_script PROGBITS         0000000000b1651a  006e651a
       0000000000000026  0000000000000000           0     0     1
  [19] .zdebug_info      PROGBITS         0000000000b16540  006e6540
       00000000000a5dce  0000000000000000           0     0     8
  [20] .zdebug_loc       PROGBITS         0000000000bbc30e  0078c30e
       000000000006db20  0000000000000000           0     0     8
  [21] .zdebug_ranges    PROGBITS         0000000000c29e2e  007f9e2e
       0000000000026244  0000000000000000           0     0     8
  [22] .note.go.buildid  NOTE             0000000000400f9c  00000f9c
       0000000000000064  0000000000000000   A       0     0     4
  [23] .symtab           SYMTAB           0000000000000000  00821000
       0000000000035b08  0000000000000018          24   274     8
  [24] .strtab           STRTAB           0000000000000000  00856b08
       000000000004aa5a  0000000000000000           0     0     1
Key to Flags:
  W (write), A (alloc), X (execute), M (merge), S (strings), I (info),
  L (link order), O (extra OS processing required), G (group), T (TLS),
  C (compressed), x (unknown), o (OS specific), E (exclude),
  l (large), p (processor specific)
```

也可以通过 `debug/elf` 包来获取 ELF 文件中的一些信息。最后我们可以从 ELF 中读取 segment 信息，它描述程序如何映射到内存中，如哪些 section 需要导入内存、采取只读模式还是读写模式、内存对其大小等：

```bash
$ readelf -lW main
Elf file type is EXEC (Executable file)
Entry point 0x463690
There are 7 program headers, starting at offset 64

Program Headers:
  Type           Offset   VirtAddr           PhysAddr           FileSiz  MemSiz   Flg Align
  PHDR           0x000040 0x0000000000400040 0x0000000000400040 0x000188 0x000188 R   0x1000
  NOTE           0x000f9c 0x0000000000400f9c 0x0000000000400f9c 0x000064 0x000064 R   0x4
  LOAD           0x000000 0x0000000000400000 0x0000000000400000 0x2e8cc4 0x2e8cc4 R E 0x1000
  LOAD           0x2e9000 0x00000000006e9000 0x00000000006e9000 0x326d43 0x326d43 R   0x1000
  LOAD           0x610000 0x0000000000a10000 0x0000000000a10000 0x0425c0 0x072128 RW  0x1000
  GNU_STACK      0x000000 0x0000000000000000 0x0000000000000000 0x000000 0x000000 RW  0x8
  LOOS+0x5041580 0x000000 0x0000000000000000 0x0000000000000000 0x000000 0x000000     0x8

 Section to Segment mapping:
  Segment Sections...
   00     
   01     .note.go.buildid 
   02     .text .note.go.buildid 
   03     .rodata .typelink .itablink .gosymtab .gopclntab 
   04     .go.buildinfo .noptrdata .data .bss .noptrbss 
   05     
   06
```

例如，我们可以通过 objdump 可以导出 .note.go.buildid 段中的信息，其中包含 Go 程序唯一的 ID。

```bash
$ objdump -s -j .go.buildinfo main
main:     file format elf64-x86-64

Contents of section .note.go.buildid:
 400f9c 04000000 53000000 04000000 476f0000  ....S.......Go..
 400fac 744d7445 7a77334f 2d747164 30787774  tMtEzw3O-tqd0xwt
 400fbc 57686478 2f667130 4148586e 68344f7a  Whdx/fq0AHXnh4Oz
 400fcc 6a6c4845 786d6b4e 632f7146 7735644d  jlHExmkNc/qFw5dM
 400fdc 54324549 4e33336b 4f2d7973 5f6c2f76  T2EIN33kO-ys_l/v
 400fec 76623171 6e56594a 636c3951 36437339  vb1qnVYJcl9Q6Cs9
 400ffc 68387900                             h8y.
```
