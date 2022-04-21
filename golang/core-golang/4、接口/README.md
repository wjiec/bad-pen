接口
------

在Go语言中，接口是一个编程规约，也是一组方法签名（函数字面量类型，不包括函数名）的集合。一个具体类型的方法集是接口方法集的超级，就代表类型实现了这个接口（不需要在语法上进行显式的声明）。**编译器会在编译时进行方法集的校验**。

最常使用的接口字面量类型是空接口（`interface{}`），由于空接口的方法集为空，所以任何类型都可以被认为实现了空接口。任意类型的实例都可以赋值或传递给空接口，包括未命名类型。



### 基本概念

接下来主要介绍

#### 接口声明

Go语言中接口分为接口字面量和接口命名类型，接口的声明使用`interface`关键字

```go
// 接口字面量
interface {
    Write([]byte) (int, error)
}

// 接口字面量的使用
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    _, _ = w.(interface {
        Write([]byte) (int, error)
    })
})

// 空接口也是一个接口字面量
interface{}

// 接口命名类型
type InterfaceName interface {
    FunctionSignature()
}

// 也可以嵌入另一个接口类型
type EmbeddedInterface interface {
    io.Reader
    io.Writer
}
```

**Go编译器在做接口匹配判断时时严格校验方法名称和方法签名（方法字面量，不包括方法名）的**。
