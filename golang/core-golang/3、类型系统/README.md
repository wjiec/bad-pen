类型系统
------------

类型系统能够让编译器在编译阶段发现大部分程序错误。



### 类型简介

类型可分为命名类型和未命名类型（又称类型字面量，复合类型）。命名类型是可以通过标识符来表示的类型，包括**Go预声明类型（简单类型）和用户自定义的类型**（使用type声明）。而未命名类型则是由预声明类型、关键字和操作符组合而成，在Go中的**复合类型**（数组、切片、字典、通道、指针、函数字面量、结构和接口）都属于未命名类型（与类型字面量，复合类型等价）。

```go
// Person 是一个命名类型
type Person struct {
	Name string
	Age  int
}

// logger 是一个使用未命名类型声明的变量
var logger struct {
	Level  int
	Format string
}

func main() {
	fmt.Printf("person => %T\n", Person{})
	fmt.Printf("logger => %T\n", logger)

	// Output:
	// 	person => main.Person
	//	logger => struct { Level int; Format string }
}
```

总结以上内容就是：

* 命名类型包含：Go语言中的简单类型（布尔、整形、浮点、复数、字符、字符串、接口）、用户自定义类型（通过`type`声明）
* 未命名类型包含：Go语言中的复合类型（数组、切片、字典、通道、指针、函数字面量、结构体、接口）

**注意以上两者都包含“接口”类型，接口也是可以匿名使用的，如下所示：**

```go
var printer interface {
    Print(string) error
}
```



#### 底层类型

在Go中所有的类型都有一个底层类型（`Underlying type`），底层类型的规则如下：

* 预声明类型和复合类型的底层类型就是他们自身（*注意不要和命名类型混在一起*）
* 自定义类型的底层类型是逐层向下查找，直到找到预声明类型或复合类型为止

```go
// 底层类型为 string
type String string
// 底层类型为 string
type UTF8String String

// 底层类型为 []string
type StringList []string
// 底层类型为 []UTF8String
type UTF8StringList []UTF8String

// 注意：StringList 和 UTF8StringList 的底层类型并不相同
```

底层类型在类型赋值和类型强制转换时会使用



#### 相同类型与类型赋值

在Go中两个类型是否相同遵循以下规则：

* 具有相同的**命名类型**（类型的名字一致）
* 具有相同结构且内部元素的类型相同的**未命名类型**
* 通过**类型别名**声明的类型（`type T1 = T2`于Go 1.9引入）
* **命名类型和未命名类型永远不相同**

变量之间在满足一定条件情况下可以**直接赋值**：

* 两个变量的**类型相同**
* 两个变量**具有相同的底层类型且至少有一个为未命名类型**
* 其中一个变量实现了**另一个接口变量**的的所有方法
* 都是通道类型且具有相同的元素类型
* 其中一个值是nil，且另一个则是指针、函数、切片、字典、通道、接口类型的变量（`var a *int; a = nil`）
* 字面值可以被赋值给相对应类型（`var a int; a = 111`）

```go
type String string
type UTF8String string

var a String
var b UTF8String

a = b // 这里不能直接赋值，应该两个都是命名类型


type StringList []string

var al []string
var bl StringList

bl = al // 这里可以直接赋值，因为 al 是未命名类型
```



#### 类型强制转换

一个变量可以通过强制类型转换赋值给另一个**底层类型相同**的变量，或者是满足一些特殊情况（比如`string`和`[]byte`、`[]rune`之间的转换）。



### 类型方法

