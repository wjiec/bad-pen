字符串本质与实现
-------------------------

字符串一般有两种类型，一种在编译时指定长度，不能修改，一种具有动态的长度，可以修改。在 Go 语言的运行时中，字符串的表示结构如下：

```go
type StringHeader {
    Data uintptr
    Len int
}
```

其中，Data 指向底层字符数组，Len 代表字符串的长度。字符串在本质上是一串字符数组，每个字符在存储时都对应一个或多个整数，这涉及字符集的编码方式。

字符串常量存储于静态存储区，其内容不可以被改变。



### 符文（rune）类型

在 Go 语言中使用符文（rune）类型来表示和区分字符串中的“字符”，rune 起始就是 int32 的别称。当使用 range 遍历字符串时，遍历的不再是单个字节，而是具体的 rune。如下所示：

```go
func main() {
	s := "Hi, 世界"

	fmt.Println("Foreach:")
	for i := 0; i < len(s); i++ {
		fmt.Printf("%2d: %#x(%T)\n", i, s[i], s[i])
	}
	// Foreach:
	// 0: 0x48(uint8)
	// 1: 0x69(uint8)
	// 2: 0x2c(uint8)
	// 3: 0x20(uint8)
	// 4: 0xe4(uint8)
	// 5: 0xb8(uint8)
	// 6: 0x96(uint8)
	// 7: 0xe7(uint8)
	// 8: 0x95(uint8)
	// 9: 0x8c(uint8)

	fmt.Println("\nRange:")
	for i, r := range s {
		fmt.Printf("%2d: %#x(%T)\n", i, r, r)
	}
	// Range:
	// 0: 0x48(int32)
	// 1: 0x69(int32)
	// 2: 0x2c(int32)
	// 3: 0x20(int32)
	// 4: 0x4e16(int32)
	// 7: 0x754c(int32)
}
```



### 字符串底层原理

字符串常量在词法解析截断最终会被标记成 StringLit 类型的 Token 并被传递到编译的下一个截断。在语法分析截断，采取递归下降的方式读取 UTF-8 支付，反引号或双引号是字符串的标识。分析的具体逻辑位于 syntax/scanner.go 文件中。



#### 字符串拼接

当对字符串执行 “+” 操作时，在编译的抽象语法树阶段具体操作的 Op 会被解析为 OADDSTR。对两个字符串常量的拼接会在语法分析截断调用 noder.sum 函数，使用 `strings.Join` 函数完成对字符串常量数组的拼接。如果涉及到字符串变量的拼接，那么其拼接操作最终是在运行时完成的。

对字符串变量的拼接，在语法分析阶段会做一些准备工作。例如在 typecheck1 函数解析赋值及字符串拼接语义时，walkexpr 函数会决定具体使用运行时的哪一个拼接函数。最终的字符串拼接都是调用的 `concatstrings` 函数，并将需要拼接的字符串通过切片传入。

```go
// concatstrings implements a Go string concatenation x+y+z+...
// The operands are passed in the slice a.
// If buf != nil, the compiler has determined that the result does not
// escape the calling function, so the string data can be stored in buf
// if small enough.
func concatstrings(buf *tmpBuf, a []string) string {
	idx := 0
	l := 0
	count := 0
	for i, x := range a {
		n := len(x)
		if n == 0 {
			continue
		}
		if l+n < l {
			throw("string concatenation too long")
		}
		l += n
		count++
		idx = i
	}
	if count == 0 {
		return ""
	}

	// If there is just one string and either it is not on the stack
	// or our result does not escape the calling frame (buf != nil),
	// then we can return that string directly.
	if count == 1 && (buf != nil || !stringDataOnStack(a[idx])) {
		return a[idx]
	}
	s, b := rawstringtmp(buf, l)
	for _, x := range a {
		copy(b, x)
		b = b[len(x):]
	}
	return s
}

func rawstringtmp(buf *tmpBuf, l int) (s string, b []byte) {
	if buf != nil && l <= len(buf) {
		b = buf[:l]
		s = slicebytetostringtmp(&b[0], len(b))
	} else {
		s, b = rawstring(l)
	}
	return
}
```



#### 字符串与字节数组的转换

字符串与字节数组之间可以想换转换，但是这个转换不是简单的指针引用，而是涉及了复制。当字符串大于或字节数组大于 32 字节时，还需要申请堆内存。因此在涉及一些密集的转换场景时，需要评估这种转换带来的性能损耗。

字节数组转换为字符串是在运行时调用了 `slicebytetostring` 函数：

```go
// slicebytetostring converts a byte slice to a string.
// It is inserted by the compiler into generated code.
// ptr is a pointer to the first element of the slice;
// n is the length of the slice.
// Buf is a fixed-size buffer for the result,
// it is not nil if the result does not escape.
func slicebytetostring(buf *tmpBuf, ptr *byte, n int) (str string) {
	if n == 0 {
		// Turns out to be a relatively common case.
		// Consider that you want to parse out data between parens in "foo()bar",
		// you find the indices and convert the subslice to string.
		return ""
	}
	if raceenabled {
		racereadrangepc(unsafe.Pointer(ptr),
			uintptr(n),
			getcallerpc(),
			funcPC(slicebytetostring))
	}
	if msanenabled {
		msanread(unsafe.Pointer(ptr), uintptr(n))
	}
	if n == 1 {
		p := unsafe.Pointer(&staticuint64s[*ptr])
		if sys.BigEndian {
			p = add(p, 7)
		}
		stringStructOf(&str).str = p
		stringStructOf(&str).len = 1
		return
	}

	var p unsafe.Pointer
	if buf != nil && n <= len(buf) {
		p = unsafe.Pointer(buf)
	} else {
		p = mallocgc(uintptr(n), nil, false)
	}
	stringStructOf(&str).str = p
	stringStructOf(&str).len = n
	memmove(p, unsafe.Pointer(ptr), uintptr(n))
	return
}
```

当字符转换为字节数组时，在运行时调用的是 `stringtoslicebyte` 函数：

```go
func stringtoslicebyte(buf *tmpBuf, s string) []byte {
	var b []byte
	if buf != nil && len(s) <= len(buf) {
		*buf = tmpBuf{}
		b = buf[:len(s)]
	} else {
		b = rawbyteslice(len(s))
	}
	copy(b, s)
	return b
}
```

