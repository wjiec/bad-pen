测试
-------



1、某些时候，我们需要为测试用例提供初始化和清理操作，但 `testing` 并没有 `setup/teardown` 机制。解决办法是自定义一个名为 `TestMain` 的函数，`go test` 会改为执行该函数，而不再是具体的测试用例

```go
func TestMain(m *testing.M) {
	// setup
	fmt.Println("setup")

	m.Run()

	// teardown
	fmt.Println("teardown")

	// setup
	// === RUN   TestAdd
	// --- PASS: TestAdd (0.00s)
	// PASS
	// teardown
}

func Add(a, b int) int {
	return a + b
}

func TestAdd(t *testing.T) {
	if Add(1, 1) != 2 {
		t.Errorf("1 + 1 != 2")
	}
}
```

