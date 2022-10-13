函数和栈
-------------

函数是程序中为了执行特定任务而存在的一系列执行代码，执行程序的过程可以看作一系列函数的调用过程。



### 函数的基本使用方式

使用函数具有减少冗余、隐藏信息、提高代码清晰度等优点。在 Go 语言中，函数可以被当做变量，并且可以作为参数传递、返回和赋值。Go 语言中的函数还具有多返回值的特点，多返回值最常用于返回 error 错误信息，从而被调用者捕获。



### 函数闭包和陷阱

闭包是一个包含了函数的入口地址及其关联的上下文环境的特殊变量。在 range 中使用闭包时需要注意循环变量的问题（循环变量的地址都是相同的，为了避免内存开销，Go 语言会复用循环变量）。



### 函数栈

在现代计算机系统中，每个线程都有一个被称为栈的内存区域，其遵循一种后入先出（LIFO，Last In First Out）的形式，增长方向从高地址到低地址。

每个函数在执行过程中都使用一块栈内存来保证返回地址、局部变量、函数参数等，我们将这一块区域称为函数的栈帧（Stack frame）。

当函数执行时，函数的参数、返回地址、局部变量会被压入栈中，当函数退出时，这些数据会被回收。维护和管理函数栈帧非常重要，对于高级编程语言来说，栈帧通常是隐藏的。



### Go 语言栈帧结构

Go 语言的函数调用栈和 C 语言有些类似，我们可以对以下函数进行分析：

```go
package main

func mul(a, b int) int {
    return a * b
}

func main() {
    mul(0xaa, 0xff)
}
```

我们可以使用命令`go tool compile -S -N -l main.go > main.S` 得到以下反汇编代码（经过删减）：

```asm
"".mul STEXT nosplit size=29 args=0x18 locals=0x0
	0x0000 00000 (main.go:3)	TEXT	"".mul(SB), NOSPLIT|ABIInternal, $0-24
	// 初始化返回值
	0x0000 00000 (main.go:3)	MOVQ	$0, "".~r2+24(SP)
	// 计算函数值
	0x0009 00009 (main.go:4)	MOVQ	"".b+16(SP), AX
	0x000e 00014 (main.go:4)	MOVQ	"".a+8(SP), CX
	0x0013 00019 (main.go:4)	IMULQ	AX, CX
	0x0017 00023 (main.go:4)	MOVQ	CX, "".~r2+24(SP)
	// 返回上一级
	0x001c 00028 (main.go:4)	RET

"".main STEXT size=75 args=0x0 locals=0x20
	0x0000 00000 (main.go:7)	TEXT	"".main(SB), ABIInternal, $32-0
	// 检查是否需要扩展栈
	0x0000 00000 (main.go:7)	MOVQ	TLS, CX
	0x0009 00009 (main.go:7)	MOVQ	(CX)(TLS*2), CX
	0x0010 00016 (main.go:7)	CMPQ	SP, 16(CX)
	0x0014 00020 (main.go:7)	JLS	68
	// 开辟栈空间
	0x0016 00022 (main.go:7)	SUBQ	$32, SP
	0x001a 00026 (main.go:7)	MOVQ	BP, 24(SP)
	0x001f 00031 (main.go:7)	LEAQ	24(SP), BP
	// 准备函数
	0x0024 00036 (main.go:8)	MOVQ	$170, (SP)
	0x002c 00044 (main.go:8)	MOVQ	$255, 8(SP)
	0x0035 00053 (main.go:8)	CALL	"".mul(SB)
	// 回收栈空间
	0x003a 00058 (main.go:9)	MOVQ	24(SP), BP
	0x003f 00063 (main.go:9)	ADDQ	$32, SP
	0x0043 00067 (main.go:9)	RET
	// 执行栈扩展
	0x0044 00068 (main.go:9)	NOP
	0x0044 00068 (main.go:7)	CALL	runtime.morestack_noctxt(SB)
	0x0049 00073 (main.go:7)	JMP	0
```

对于以上汇编代码，针对栈的部分，我们可以看到以下代码：

* `"".main(SB), ABIInternal, $32-0`：最后的 32 表示当前栈帧需要分配的字节数，随后的 0 表示参数和返回值的字节数
* `SUBQ $32, SP`：表示开辟 32 字节的栈空间出来（栈从高地址往低地址生长）
* `MOVQ BP, 24(SP)`：**将当前 BP 寄存器的值保存到栈的底部**
* `LEAQ 24(SP), BP`：将 BP 寄存器指向当前栈帧的栈底（去掉保存前 BP 寄存器的 8 个字节）

此时我们有以下结构：

```go
+..................+
| RETURN ADDRESS   |
+..................+ +32(SP)             ───
| PREVIOUS BP      |                      |
+──────────────────+ +24(SP) [CURRENT BP] |
| RETURN VALUE     |                      |
+──────────────────+ +16(SP)              | MAIN FRAME
| SECOND ARGUMENT  |                      |
+──────────────────+ +8(SP)               |
| FIRST ARGUMENT   |                      |
+..................+ +0(SP)              ───
| RETURN MAIN ADDR |
+..................+
```



### 堆栈信息

