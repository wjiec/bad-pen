变量及数据
----------------



1、字符串操作通常在堆上分配内存，这会对 Web 等高并发应用造成较大影响，因为在字符串操作过程中会有大量的字符串对象要做垃圾回收。**建议使用 `[]byte` 缓存池，或在栈上自行拼装等方式来实现 zero-garbage**

```go
//
// strings.go
//

func Join(ss ...string) string {
	return strings.Join(ss, "")
}

func ConcatMulti(ss ...string) string {
	s := ss[15]
	s += ss[14]
	s += ss[13]
	s += ss[12]
	s += ss[11]
	s += ss[10]
	s += ss[9]
	s += ss[8]
	s += ss[7]
	s += ss[6]
	s += ss[5]
	s += ss[4]
	s += ss[3]
	s += ss[2]
	s += ss[1]
	s += ss[0]

	return s
}

func ConcatSingle(ss ...string) string {
	return ss[15] + ss[14] + ss[13] + ss[12] + ss[11] + ss[10] + ss[9] + ss[8] +
		ss[7] + ss[6] + ss[5] + ss[4] + ss[3] + ss[2] + ss[1] + ss[0]
}

//
// strings_test.go
//

func BenchmarkJoin(b *testing.B) {
	s := make([]string, 16)
	for i := 0; i < len(s); i++ {
		s[i] = "a"
		if i != 0 {
			s[i] += s[i-1]
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Join(s...)
	}

	// BenchmarkJoin
	// BenchmarkJoin-8           	 4379310	       252.1 ns/op
}

func BenchmarkConcatMulti(b *testing.B) {
	s := make([]string, 16)
	for i := 0; i < len(s); i++ {
		s[i] = "a"
		if i != 0 {
			s[i] += s[i-1]
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcatMulti(s...)
	}

	// BenchmarkConcatMulti
	// BenchmarkConcatMulti-8    	 1000000	      1438 ns/op
}

func BenchmarkConcatSingle(b *testing.B) {
	s := make([]string, 16)
	for i := 0; i < len(s); i++ {
		s[i] = "a"
		if i != 0 {
			s[i] += s[i-1]
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcatSingle(s...)
	}

	// BenchmarkConcatSingle
	// BenchmarkConcatSingle-8   	 5454232	       241.8 ns/op
}
```

