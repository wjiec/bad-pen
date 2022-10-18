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

在 Go 程序 panic 时，会输出一系列堆栈信息，这是调试 Go 语言的一种基本方法。我们使用如下示例进行测试：

```go
package main

func trace(b byte, arr []int, f float64) {
	panic("panic trace")
}

func main() {
	trace(0, []int{1, 2, 3}, 4.0)
}
```

我们可以通过 `go run -gcflags="-l" main.go` 来禁止函数内联并输出当前协程所在的堆栈：

```plain
panic: panic trace									// 程序终止的原因

goroutine 1 [running]:								// 触发 panic 的协程信息
main.trace(0x70, {0x404719, 0x60, 0x0}, 0x0)		// 当前协程的函数调用链信息
	/go/src/main.go:5 +0x27							// 当前函数所在的文件以及行号，以及下一个指令的偏移量
main.main()
	/go/src/main.go:9 +0x5d
exit status 2
```

Go 语言可以通过配置 GOTRACEBACK 环境变量在程序异常终止时生成 coredump 文件（可以用过 dlv 或者 gdb 等高级调试工具进行分析调试）。



### 栈扩容与栈转移原理

在 Go 语言中，每个协程都有一个栈，并且在 Go 1.4 之后，每个栈的大小在初始化的时候都是 2KB，在 64 位系统中最大可以扩容到 1GiB，在 32 位系统中则是 250MiB。在 Go 语言中，栈的大小不用开发者手动调整，都是在运行时实现的。栈的管理有两个重要问题：触发扩容的时机以及调整的方式。

触发扩容的时机时在函数头阶段，由编译器自动插入的判断指令，如果满足一定条件则需要对栈进行扩容。其核心逻辑依赖于 g 结构体中的 stack 结构体以及相关的内部字段：

```go
//
// runtime/runtime2.go
//

type g struct {
	// Stack parameters.
	// stack describes the actual stack memory: [stack.lo, stack.hi).
	// stackguard0 is the stack pointer compared in the Go stack growth prologue.
	// It is stack.lo+StackGuard normally, but can be StackPreempt to trigger a preemption.
	// stackguard1 is the stack pointer compared in the C stack growth prologue.
	// It is stack.lo+StackGuard on g0 and gsignal stacks.
	// It is ~0 on other goroutine stacks, to trigger a call to morestackc (and crash).
	stack       stack   // offset known to runtime/cgo
	stackguard0 uintptr // offset known to liblink
	stackguard1 uintptr // offset known to liblink
    
    // ...
}

// Stack describes a Go execution stack.
// The bounds of the stack are exactly [lo, hi),
// with no implicit data structures on either side.
type stack struct {
	lo uintptr
	hi uintptr
}
```

 在函数头插入的指令如下：

```go
"".main STEXT size=75 args=0x0 locals=0x20
	0x0000 00000 (main.go:7)	TEXT	"".main(SB), ABIInternal, $32-0
	// 检查是否需要扩展栈
	0x0000 00000 (main.go:7)	MOVQ	TLS, CX
	0x0009 00009 (main.go:7)	MOVQ	(CX)(TLS*2), CX
	0x0010 00016 (main.go:7)	CMPQ	SP, 16(CX)
	0x0014 00020 (main.go:7)	JLS	68

	// ...

	// 执行栈扩展
	0x0044 00068 (main.go:9)	NOP
	0x0044 00068 (main.go:7)	CALL	runtime.morestack_noctxt(SB)
	0x0049 00073 (main.go:7)	JMP	0
```

在代码中，将会取出 TLS（Thread Local Storage）中的 g 结构体，将结构体中的 stackgrard0 与 SP 寄存器相比较，如果满足 `SP < stackguard0` 则执行 `runtime.morestack_noctxt` 进行扩容：

```go
//
// runtime/asm_amd64.s
//

// morestack but not preserving ctxt.
TEXT runtime·morestack_noctxt(SB),NOSPLIT,$0
	MOVL	$0, DX
	JMP	runtime·morestack(SB)

// Called during function prolog when more stack is needed.
//
// The traceback routines see morestack on a g0 as being
// the top of a stack (for example, morestack calling newstack
// calling the scheduler calling newm calling gc), so we must
// record an argument size. For that purpose, it has no arguments.
TEXT runtime·morestack(SB),NOSPLIT,$0-0
	// Cannot grow scheduler stack (m->g0).
	get_tls(CX)
	MOVQ	g(CX), BX
	MOVQ	g_m(BX), BX
	MOVQ	m_g0(BX), SI
	CMPQ	g(CX), SI
	JNE	3(PC)
	CALL	runtime·badmorestackg0(SB)
	CALL	runtime·abort(SB)

	// Cannot grow signal stack (m->gsignal).
	MOVQ	m_gsignal(BX), SI
	CMPQ	g(CX), SI
	JNE	3(PC)
	CALL	runtime·badmorestackgsignal(SB)
	CALL	runtime·abort(SB)

	// Called from f.
	// Set m->morebuf to f's caller.
	NOP	SP	// tell vet SP changed - stop checking offsets
	MOVQ	8(SP), AX	// f's caller's PC
	MOVQ	AX, (m_morebuf+gobuf_pc)(BX)
	LEAQ	16(SP), AX	// f's caller's SP
	MOVQ	AX, (m_morebuf+gobuf_sp)(BX)
	get_tls(CX)
	MOVQ	g(CX), SI
	MOVQ	SI, (m_morebuf+gobuf_g)(BX)

	// Set g->sched to context in f.
	MOVQ	0(SP), AX // f's PC
	MOVQ	AX, (g_sched+gobuf_pc)(SI)
	LEAQ	8(SP), AX // f's SP
	MOVQ	AX, (g_sched+gobuf_sp)(SI)
	MOVQ	BP, (g_sched+gobuf_bp)(SI)
	MOVQ	DX, (g_sched+gobuf_ctxt)(SI)

	// Call newstack on m->g0's stack.
	MOVQ	m_g0(BX), BX
	MOVQ	BX, g(CX)
	MOVQ	(g_sched+gobuf_sp)(BX), SP
	CALL	runtime·newstack(SB)
	CALL	runtime·abort(SB)	// crash if newstack returns
	RET
```

在最后调用的 `runtime.newstack` 方法首先通过栈底地址与栈顶地址得到旧栈的大小，将其扩大一倍得到新栈的大小：

```go
//
// runtime/stack.go
//

// Called from runtime·morestack when more stack is needed.
// Allocate larger stack and relocate to new stack.
// Stack growth is multiplicative, for constant amortized cost.
//
// g->atomicstatus will be Grunning or Gscanrunning upon entry.
// If the scheduler is trying to stop this g, then it will set preemptStop.
//
// This must be nowritebarrierrec because it can be called as part of
// stack growth from other nowritebarrierrec functions, but the
// compiler doesn't check this.
//
//go:nowritebarrierrec
func newstack() {
	// ...

	// Allocate a bigger segment and move the stack.
	oldsize := gp.stack.hi - gp.stack.lo
	newsize := oldsize * 2

	if gp.stackguard0 == stackForceMove {
		// Forced stack movement used for debugging.
		// Don't double the stack (or we may quickly run out
		// if this is done repeatedly).
		newsize = oldsize
	}

	if newsize > maxstacksize || newsize > maxstackceiling {
		if maxstacksize < maxstackceiling {
			print("runtime: goroutine stack exceeds ", maxstacksize, "-byte limit\n")
		} else {
			print("runtime: goroutine stack exceeds ", maxstackceiling, "-byte limit\n")
		}
		print("runtime: sp=", hex(sp), " stack=[", hex(gp.stack.lo), ", ", hex(gp.stack.hi), "]\n")
		throw("stack overflow")
	}

	// The concurrent GC will not scan the stack while we are doing the copy since
	// the gp is in a Gcopystack status.
	copystack(gp, newsize)
	if stackDebug >= 1 {
		print("stack grow done\n")
	}
	casgstatus(gp, _Gcopystack, _Grunning)
	gogo(&gp.sched)
}
```

栈扩容中最重要的就是将旧栈中的内容转移到新栈中，栈的复制并不像直接复制内存那么简单，如果栈中包含了引用栈中其他变量的指针，那么该指针也需要对应到新栈中的地址。其中 `runtime.copystack` 函数会遍历新栈上所有的栈帧信息，并遍历其中所有可能有指针的位置。一旦发现指针指向旧栈，就会调整当前的函数指针使其指向新栈。



### 栈调试

Go 语言在源码级别提供了栈相关的多种级别的调试、用户调试栈的扩容以及栈的分配等等。如果需要使用这些变量，则需要直接修改 Go 的源码并重新编译。

```go
//
// runtime/stack.go
//

const (
	// stackDebug == 0: no logging
	//            == 1: logging of per-stack operations
	//            == 2: logging of per-frame operations
	//            == 3: logging of per-word updates
	//            == 4: logging of per-word reads
	stackDebug       = 0
	stackFromSystem  = 0 // allocate stacks from system memory instead of the heap
	stackFaultOnFree = 0 // old stacks are mapped noaccess to detect use after free
	stackPoisonCopy  = 0 // fill stack that should not be accessed with garbage, to detect bad dereferences during copy
	stackNoCache     = 0 // disable per-P small stack caches

	// check the BP links during traceback.
	debugCheckBP = false
)
```

同时我们也可以使用 `runtime/debug.PrintStack` 方法来打印当前时刻的堆栈信息。**在调试时，一定要注意取消编译器的优化并避免函数内敛，否则不会得到预期的结果。**

我们还可以通过标准库 `pprof` 来获取当前时刻的栈信息。需要注意其展现形式与之前的堆栈信息略有不同。利用 pprof 的协程栈调试，可以非常方便地分析是否发生协程泄露、当前程序使用最多的函数是什么，并分析 CPU 的瓶颈、性能可视化等。
