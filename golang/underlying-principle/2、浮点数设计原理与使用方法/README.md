2、浮点数设计原理与使用方法
-------------------------------------------

浮点数能够在程序中高效地表示和计算小数，但是在表示和计算的过程中肯呢过丢失精度。我们必须深入理解浮点数在计算机中的存储方式及性质，才能正确处理数字的计算问题。



### 定点数和浮点数

计算机通过二进制的形式存储数据，然而大多数小数表示出二进制后是近似且无限的。在有限的内存空间中无法表达无限的结果，计算机必须有其他的机制来处理小数的存储与计算。

最简单的表示小数的方法是**定点表示法**，即用固定的大小来表示整数，剩余部分表示小数。这种方式在某些场景下可能很适用，但并不适用于所有场景。Go 语言与其他很多语言（C、C++、Python）一样，都采用 IEEE-754 浮点数标准来存储小数。在该标准中规定了浮点数的存储、计算、四舍五入、异常处理等一系列规则。



### IEEE-754 浮点数标准

该规范使用以 2 为底的指数来表示小数，其中最开头的一位为符号位，1 表示负数，0表示正数。符号位之后为指数位，单精度为 8 位，双精度为 11 位。指数位存储了指数加上偏移量的值（单精度为 127，双精度为 1023），这是为了表达负数而设计的（比如 -4 在单精度中表示为 127 - 4 = 123）。剩下的都是小数位，小数位中存储系数的准确值或最接近的值，是 0 ~ 1 之间的数。

在小数位中的每一位代表的都是以 2 为底的幂，并且指数依次减少 1。Go 语言标准库的 math 包提供了许多有用的计算函数，比如我们可以使用如下方法输出浮点数中的每一个比特：

```go
func DumpFloat32(number float32) {
	bits := math.Float32bits(number)
	binary := fmt.Sprintf("%.32b", bits)

	// sign: 1b, exponent: 8b, mantissa: 23

	fmt.Printf("Number: %f\n", number)
	fmt.Printf("Pattern: %s | %s %s | %s %s %s %s %s %s\n",
		binary[0:1],              // sign
		binary[1:5], binary[5:9], // exponent
		binary[9:12], binary[12:16], binary[16:20], // mantissa
		binary[20:24], binary[24:28], binary[28:32], // mantissa
	)

	sign := (bits & (1 << 31)) >> 31          // shift sign
	var exponent = int32((bits >> 23) & 0xff) // remove mantissa

	var mantissa float64
	for index, bit := range binary[9:32] {
		if bit == '1' {
			mantissa += 1 / math.Pow(2, float64(index+1))
		}
	}

	value := (1 + mantissa) * math.Pow(2, float64(exponent-127))
	fmt.Printf("Sign: %d  Exponnt: %d (%d)  Mantissa: %f  Value: %f\n",
		sign,
		exponent, exponent-127,
		mantissa, value,
	)

	fmt.Println()
}

func main() {
	DumpFloat32(16.3472)
    // Number: 16.347200
    // Pattern: 0 | 1000 0011 | 000 0010 1100 0111 0001 0001
    // Sign: 0  Exponnt: 131 (4)  Mantissa: 0.021700  Value: 16.347200

	DumpFloat32(-0.12560)
    // Number: -0.125600
    // Pattern: 1 | 0111 1100 | 000 0000 1001 1101 0100 1001
    // Sign: 1  Exponnt: 124 (-3)  Mantissa: 0.004800  Value: 0.125600
}
```



### 浮点数精度

在一个范围内，将 d 位十进制数（按照科学计数法表达）转换为二进制数，再将二进制数转换为 d 位十进制数，如果数据转换不发生损失，则意味着在此范围内有 d 位精度。

理论表明，单精度浮点数 float32 的精度为 6~8 位，双精度浮点数 float64 的精度为 15~17 位。 浮点数不仅在表示时会丢失精度，在计算过程中也可能丢失精度。浮点数的输出是通过调用 `strconv.FormatFloat` 方法，借助 Grisu3 算法快速并准确地格式化浮点数，该算法的速度是普通高精度算法的 4 倍。



### 多精度浮点数与 math/big 库

在一些特定领域，如果需要更高精度的存储和计算时可以使用 math/big 标准库，其中提供了处理大数的三种数据类型：

* `big.Int`：核心思想就是用 uint 切片来存储大整数，其可以容纳超过 int64 的数字，甚至可以认为是无限扩容的
* `big.Float`：核心思路就是把服务店转换为大整数运算，其仍然会损失精度（可通过 prec 参数控制）
* `big.Rat`：核心思想就是将有理数的运算转换为分数的运算，这将永远不会损失精度
