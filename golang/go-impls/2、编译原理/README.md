编译原理
-------------

想了解 Go 语言的实现原理，理解其编译过程就是一个无法绕过的事情。



### 编译过程

想深入了解 Go 语言的变异过程，需要提前了解编译过程中涉及的一些术语和专业知识。

#### 抽象语法树

抽象语法树（Abstract Syntax Tree，AST）是源代码语法结构的一种抽象表示，它用树状的方法表示编程语言的语法结构。作为编译器常用的数据结构，抽象语法树抹去了源代码中一些不重要的字符，如空格、分号、括号等。

抽象语法树辉辅助编译器进行语义分析，我们可以用它来确定语法正确的程序中是否存在一些类型不匹配的问题。

#### 静态单赋值

静态单赋值（Static Single Assignment，SSA）是中间代码的一种特征，如果中间代码具有 SSA 特性，那么每个变量就只会被赋值一次。

SSA 的主要作用是对代码进行优化，所以它是编译器后端的一部分。

#### 指令集

x86 是目前比较常用的指令集，除 x86 外，还有 ARM 等指令集。指令集有复杂指令集（Complex Instruction Set Computer，CISC）和精简指令集（Reduced Instruction Set Computer，RISC）是两种遵循不同设计理念的指令集。

* 复杂指令集：通过增加指令的类型减少需要执行的指令数量
* 精简指令集：使用更少的指令类型完成目标计算任务



### 编译四阶段

编译器前段一般承担着词法分析、语法分析、类型检查和中间代码生成几部分工作，而编译器后端主要负责目标代码生成和优化，也就是将中间代码翻译生目标及其能够运行的二进制机器码。

Go 语言的编译器在逻辑上可以分成 4 个步骤：词法分析与语法分析、类型检查、中间代码生成和最后的机器代码生成。

#### 词法分析与语法分析

词法分析的作用就是解析源代码文件，它是将文件中的字符串序列转换成 Token 序列，方便后续的处理接解析。我们一般把执行词法分析的程序称为词法分析器（Lexer）。

语法分析的输入是词法分析器输出的 Token 序列。语法分析器会按照顺序解析 Token 序列，按照编程语言定义好的文法（Grammar）自下而上或自上而下地将 Token 序列转换为一棵抽象语法树。

#### 类型检查

通过遍历整棵抽象语法树，我们在每个节点上都会验证当前子树的类型，以保证节点不存在类型错误。类型检查阶段不止会验证节点的类型，还会展写和改写一些内置函数。

#### 中间代码生成

在经过类型检查之后，可以认为当前文件中的代码就不存在语法和类型错误了，Go 语言的编译器就会将输入的抽象语法树转换成中间代码。编译器会通过 `cmd/compile/internal/gc.compileFuncttions` 编译整个 Go 语言项目中的全部函数（通过队列并发执行）。

由于 Go 语言编译器的中间代码使用了 SSA 的特性，所以在该阶段编译器可以分析出代码中的无用变量和片段并对代码进行优化。

#### 机器码生成

Go 语言源代码的 `src/cmd/compile/internal` 目录中包含了很多机器码生成相关的包，不同类型的 CPU 使用了不同的包生成机器码。

#### 编译流程

Go 语言的编译从 `src/cmd/compile/main.go` 文件开始，然后进入到 `src/cmd/compile/internal/gc.Main` 函数中，随后会调用 `cmd/compile/internal/noder/noder.LoadPackage` 方法对输入文件进行词法分析与语法分析，得到对应的抽象语法树。



### 词法分析和语法分析

该过程将原本机器认为无序的源文件转换成更容易理解、分析并且结构化的抽象语法树。

#### 词法分析

为了让机器能够理解源代码，需要做的第一件事就是将字符串分组，这个过程被称为词法分析（Lexical analysis），这是将字符串序列转换为 Token 序列的过程。

词法分析作为具有固定模式的任务，就有了 lex 这种专门用于生成词法分析器的工具。我们可以通过以下内容生成一个简易的 Go 词法分析器：

```lex
%{
#include <stdio.h>
%}

%%
package         printf("PACKAGE ");
import          printf("IMPORT ");
\.              printf("DOT ");
\{              printf("L_BRACE ");
\}              printf("R_BRACE ");
\(              printf("L_PAREN ");
\)              printf("R_PAREN ");
\"              printf("QUOTE ");
\n              printf("\n");
[0-9]+          printf("NUMBER ");
[a-zA-Z_]+      printf("IDENT ");
%%
```

然后我们在终端中执行 `lex go.l` 将其展开为 C 语言代码，并通过命令 `gcc lex.yy.c -o lexier -ll` 将其编译为二进制文件。我们将下面的 Go 代码作为输入传递给词法分析器：

```go
package main
 
import (
    "fmt"
)

func main() {
    fmt.Println("Hello world!")
}
```

执行命令：`cat main.go | lexier` 可以看到输出如下所示：

```plaintext
PACKAGE  IDENT 

IMPORT  L_PAREN 
    QUOTE IDENT QUOTE 
R_PAREN 

IDENT  IDENT L_PAREN R_PAREN  L_BRACE 
    IDENT DOT IDENT L_PAREN QUOTE IDENT  IDENT !QUOTE R_PAREN 
R_BRACE
```

#### Go 语言中的词法分析

Go 语言的词法分析是通过 `src/cmd/compile/internal/syntax/scanner.scanner` 结构体实现的，这个结构体会持有当前扫描的数据源文件、启用的模式和当前被扫描到 Token：

```go
type scanner struct {
	source
	mode   uint
	nlsemi bool // if set '\n' and EOF translate to ';'

	// current token, valid after calling next()
	line, col uint
	blank     bool // line is blank up to col
	tok       token
	lit       string   // valid if tok is _Name, _Literal, or _Semi ("semicolon", "newline", or "EOF"); may be malformed if bad is true
	bad       bool     // valid if tok is _Literal, true if a syntax error occurred, lit may be malformed
	kind      LitKind  // valid if tok is _Literal
	op        Operator // valid if tok is _Operator, _Star, _AssignOp, or _IncOp
	prec      int      // valid if tok is _Operator, _Star, _AssignOp, or _IncOp
}
```

#### 语法分析

语法分析的过程会使用自顶向下或自底向上的方式进行推导。

##### 文法（Grammar）

上下文无关文法是用来形式化、精确描述某种编程语言的工具，我们能够通过文法定义一种语言的语法，它包含一系列用于转化字符串的生产规则（Production Rule）。上下文无关文法中的每一项生产规则都会将规则左侧的非终结符转换成右侧的字符串。

##### lookahead

当不同生产规则发生冲突时，分析器需要预读一些 Token 判断当前应该用什么生产规则对输入流进行展开或者归约。

#### Go 中的语法分析

Go 语言的分析器使用 LALR 的文法来解析词法分析过程中产生的 Token 序列，最右推导加向前查看构成了 Go 语言分析器的基本原理。

Go 语言的词法分析器：`src/cmd/compile/internal/syntax.scanner`

Go 语言的语法分析器：`src/cmd/compile/internal/syntax.parser`



### 类型检查

对于类型检查，首先我们需要先理解几个术语：强类型、弱类型、静态类型、动态类型

#### 强弱类型

对于强弱类型，我们一般有以下结论：

* 强类型的编程语言在编译期间会有更严格的类型限制，也就是编译器会在编译期间发现变量赋值、返回值和函数调用时的类型错误。
* 弱类型的编程语言出现类型错误时可能会在运行时进行隐式类型转换，这可能会造成运行时错误。

为了方便沟通和分类，应该把更多注意力放到下面的问题上：

* 类型的转换是显式的还是隐式的？
* 编译器会帮助我们推断变量的类型么？

#### 静态类型和动态类型

对于这两种类型，更正确的表述是：使用**静态类型检查**和**动态类型检查**的语言。

静态类型检查是基于对源代码的分析来确定运行程序类型安全的过程，静态类型检查能够帮助开发者在编译期间发现程序中出现的类型错误。

动态类型检查是在运行时确定程序类型安全的过程，它需要编程语言在编译时为所有的对象加入类型标签等信息，运行时可以使用这些存储的类型信息来实现动态派发、向下转型、反射以及其他特性。

#### Go语言中的类型检查

Go语言的编译器不仅使用静态类型检查来保证程序运行的类型安全，还会在编程器件引入类型信息，让工程师能够使用反射来判断参数和变量的类型。在 Go 语言中，主要的实现逻辑位于`src/cmd/compile/internal/gc/typecheck.go` 文件中的以下方法：

* `gc.typecheck`：负责检查常量、类型、函数声明以及变量赋值语句的类型
* `gc.typecheck1`：核心逻辑

