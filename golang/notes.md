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



3、为避免版本不一致等情况发生，可添加“import comment”，让编译器检查导入路径是否与该注释一致

```go
package xhttp // import "github.com/wjiec/xhttp"

func Run() {
    // ...
}
```



4、内置的 `print/println` 是输出到 `stderr` 的

```go
// The print built-in function formats its arguments in an
// implementation-specific way and writes the result to standard error.
// Print is useful for bootstrapping and debugging; it is not guaranteed
// to stay in the language.
func print(args ...Type)

// The println built-in function formats its arguments in an
// implementation-specific way and writes the result to standard error.
// Spaces are always added between arguments and a newline is appended.
// Println is useful for bootstrapping and debugging; it is not guaranteed
// to stay in the language.
func println(args ...Type)
```



5、当发布时，参数 `-ldflags "-w -s"` 会让链接器剔除符号表和调试信息，除能减小可执行文件大小外，还可稍稍增加反汇编的难度。还可以借助更专业的工具（比如 `upx`）对可执行文件进行瘦身



6、可以使用 `go generate` 命令扫描源码文件，找出所有 `go:generate` 注释并提取其中的命令并执行

* 命令必须放在 `.go` 源文件中
* 命令必须以 `//go:generate` 开头（双斜线后不能有空格）
* 每个文件可以有多条 `generate` 命令
* 命令支持环境变量
* 必须显式执行 `go generate` 命令
* 按文件名顺序提取命令并执行
* 串行执行，出错后终止后续命令的执行

```go
package main

//go:generate pwd

func main() {}

// $ go generate
// /workspace/gonotes
```

