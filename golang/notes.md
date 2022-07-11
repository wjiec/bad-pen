注意事项
------------



1、`return` 语句并不意味着 `RET` 指令，而仅仅是更新返回值而已

```go
func apple(v int) (int, error) {
    defer a()
    defer b()
    
    return 1, nil // MOVQ r1 = 1
    			  // MOVQ r2 = nil
    
    			  // CALL defer
    			  // RET
}
```



2、`defer` 与 `go` 关键字也会因“延迟执行”而立即计算并复制执行参数

```go
var y int

func Counter() int {
	y++
	return y
}

func main() {
	x := 100

	go func(x, y int) {
		time.Sleep(time.Second)
		fmt.Printf("goroutine: x = %d, y = %d\n", x, y)
		// goroutine: x = 100, y = 1
	}(x, Counter())

	x += 100
	defer func(x, y int) {
		fmt.Printf("defer: x = %d, y = %d\n", x, y)
		// defer: x = 200, y = 2
	}(x, Counter())

	x += 100
	fmt.Printf("main: x = %d, y = %d\n", x, Counter())
	// main: x = 300, y = 3

	time.Sleep(2 * time.Second)
}
```

