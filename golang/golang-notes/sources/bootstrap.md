引导程序
------------

编译好的可执行文件的真正入口并不是我们所写的 `main.main` 函数，因为编译器总是会插入一段引导代码，完成诸如命令行参数、运行时初始化等工作，然后才会进入用户逻辑。

我们准备如下一段程序，编译后通过 GDB 进行调试以找到程序的真正入口位置：

```go
// main.go

package main

func main() {}
```

然后我们使用命令 `go build -gcflags="-N -l" -o bootstrap main.go` 来编译它。`-N` 表示禁止优化，`-l` 表示禁止内联。接下来我们就可以使用 GDB 调试这个可执行程序：

```plain
$ gdb ./bootstrap
(gdb) info files 
Symbols from "/workspace/bootstrap/bootstrap".
Local exec file:
	`/workspace/bootstrap/bootstrap', file type elf64-x86-64.
	Entry point: 0x453b60
	0x0000000000401000 - 0x00000000004553f0 is .text
	0x0000000000456000 - 0x00000000004784b8 is .rodata
	0x0000000000478640 - 0x0000000000478900 is .typelink
	0x0000000000478900 - 0x0000000000478908 is .itablink
	0x0000000000478908 - 0x0000000000478908 is .gosymtab
	0x0000000000478920 - 0x00000000004b7d48 is .gopclntab
	0x00000000004b8000 - 0x00000000004b8020 is .go.buildinfo
	0x00000000004b8020 - 0x00000000004b91a0 is .noptrdata
	0x00000000004b91a0 - 0x00000000004bb390 is .data
	0x00000000004bb3a0 - 0x00000000004e9ec8 is .bss
	0x00000000004e9ee0 - 0x00000000004ef200 is .noptrbss
	0x0000000000400f9c - 0x0000000000401000 is .note.go.buildid

(gdb) b *0x453b60
Breakpoint 1 at 0x453b60: file /usr/local/go/src/runtime/rt0_linux_amd64.s, line 8.

(gdb) info break
Num     Type           Disp Enb Address            What
1       breakpoint     keep y   0x0000000000453b60 in _rt0_amd64_linux at /usr/local/go/src/runtime/rt0_linux_amd64.s:8
```



打开 `/usr/local/go/src/runtime/rt0_linux_amd64.s` 文件，我们可以看到如下代码：

```asm
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"

TEXT _rt0_amd64_linux(SB),NOSPLIT,$-8
	JMP	_rt0_amd64(SB)

TEXT _rt0_amd64_linux_lib(SB),NOSPLIT,$0
	JMP	_rt0_amd64_lib(SB)
```

而在 `/usr/local/go/src/runtime` 目录下有很多 `rt_*.s` 文件，分别代表不同系统和不同架构下的入口：

```bash
$ ls /usr/local/go/src/runtime/rt0_*.s
rt0_aix_ppc64.s        rt0_freebsd_amd64.s  rt0_linux_arm64.s    rt0_netbsd_amd64.s    rt0_plan9_amd64.s
rt0_android_386.s      rt0_freebsd_arm64.s  rt0_linux_arm.s      rt0_netbsd_arm64.s    rt0_plan9_arm.s
rt0_android_amd64.s    rt0_freebsd_arm.s    rt0_linux_mips64x.s  rt0_netbsd_arm.s      rt0_solaris_amd64.s
rt0_android_arm64.s    rt0_illumos_amd64.s  rt0_linux_mipsx.s    rt0_openbsd_386.s     rt0_windows_386.s
rt0_android_arm.s      rt0_ios_amd64.s      rt0_linux_ppc64le.s  rt0_openbsd_amd64.s   rt0_windows_amd64.s
rt0_darwin_amd64.s     rt0_ios_arm64.s      rt0_linux_ppc64.s    rt0_openbsd_arm64.s   rt0_windows_arm64.s
rt0_darwin_arm64.s     rt0_js_wasm.s        rt0_linux_riscv64.s  rt0_openbsd_arm.s     rt0_windows_arm.s
rt0_dragonfly_amd64.s  rt0_linux_386.s      rt0_linux_s390x.s    rt0_openbsd_mips64.s
rt0_freebsd_386.s      rt0_linux_amd64.s    rt0_netbsd_386.s     rt0_plan9_386.s
```



`_rt0_amd64_linux` 是一个包装入口，实际会跳转到 `_rt0_amd64(SB)` 位置，该符号定义于 `/usr/local/go/src/runtime/asm_amd64.s`：

```asm
// _rt0_amd64 is common startup code for most amd64 systems when using
 // internal linking. This is the entry point for the program from the
 // kernel for an ordinary -buildmode=exe program. The stack holds the
 // number of arguments and the C-style argv.
 TEXT _rt0_amd64(SB),NOSPLIT,$-8
   MOVQ  0(SP), DI // argc
   LEAQ  8(SP), SI // argv
   JMP runtime·rt0_go(SB)
```



在经过简单的初始化之后，就会跳转到同文件中的 `runtime.rt0_go` 位置继续执行：

```asm
TEXT runtime·rt0_go(SB),NOSPLIT|TOPFRAME,$0
	// ...
	CALL	runtime·args(SB)
	CALL	runtime·osinit(SB)
	CALL	runtime·schedinit(SB)

	// create a new goroutine to start program
	MOVQ	$runtime·mainPC(SB), AX		// entry
	// ...
	
	// start this M
	CALL	runtime·mstart(SB)
	// ...
```

至此，由汇编针对各平台实现的引导过程全部完成，后续控制权将会被用户和 runtime 接管。

