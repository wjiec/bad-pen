调试源代码
----------------

想要理解 Go 语言的实现原理，动手实践是必不可少的工作，也就是调试 Go 语言源代码



### 编译 Go 语言源代码

我们可以进入 Go 语言源代码的 `src` 目录，通过执行 `./make.bash` 来编译生成 Go 语言的日净值文件以及相关工具链。

```bash
git clone https://github.com/golang/go.git

cd go/src

bash make.bash
```



### 中间代码

Go 语言编译器的中间代码具有静态单赋值的特性，我们可以通过 `GOSSAFUNC` 来获取指定函数在编译时经历的所有过程：

```bash
GOSSAFUNC=main go build main.go
```

上述命令会生成一个 `ssa.html` 文件，打开该文件后就能看到汇编代码优化的每一个步骤。

