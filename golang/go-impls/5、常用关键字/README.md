常用关键字
---------------

关键字具有非常特殊的含义，它们是编程语言对外提供的接口的一部分。



### for 和 range

除了经典的 for 三段式循环外，Go 语言还引入了另一个关键字 range 帮助我们快速遍历数组、切片、哈希表和 channel 等集合类型。

#### 经典循环

经典循环在编译器看起来是一个 `OFOR` 类型的节点，该节点由以下 4 部分组成：

* 初始化循环的 `Ninit`
* 循环的继续条件 `Left`
* 循环体结束时执行的 `Right`
* 循环体 `NBody`

#### 范围循环

在编译器期间，编译器会将所有 for-range 循环变为经典循环。节点的转换过程发生在中间代码生成阶段，所有的 for-range 循环都会被编译器转换成不包含复杂结构、只包含基本表达式的语句。

##### 数组与切片

**对于所有的 range 循环，Go 语言都会在编译器将原切片或者数组赋值给一个新变量（发生了复制）**

##### 哈希表

在遍历哈希表时，编译器会使用 `runtime.mapiterinit` 和 `runtime.mapiternext` 两个运行时函数重写原始的 for-range 循环。

##### 字符串

遍历字符串的过程与遍历数组、切片和哈希表非常相似，只是在遍历时会获取字符串中的索引和对应字节并将其转换成 rune 类型。

##### channel

该循环会使用 `<-ch` 从 channel 中取出待处理的值，这个操作会调用 `runtime.chanrecv2` 并阻塞当前协程，当 `runtime.chanrecv2` 返回时会根据布尔值判断是否需要跳出循环。



### select

Go 语言中的 `select` 能够让 Goroutine 同时等待多个 channel 可读或者可写。在 Go 语言中使用 `select` 控制结构时，我们有：

* `select` 能在 channel 上进行非阻塞的收发操作（default 子句）
* `select` 在遇到多个 channel 同时响应时，会随机选择执行一个分支

#### 实现原理

`select` 语句在编译期间会被转换成 `OSELECT` 节点，在生成中间代码期间，会根据 `select` 中 `case` 的不同对控制语句进行优化：

* 不存在任何 case
* 只存在一个 case
* 存在两个 case，其中一个是 default
* 存在多个 case

##### 直接阻塞

当 `select` 结构中不包含任何 case，编译器会将 `select {}` 语句直接转换成调用 `runtime.block` 函数。此时 Goroutine 进入无法被唤醒的永久休眠状态。

##### 单一 channel

如果当前 `select` 控制结构中只包含一个 case，那么编译器会将 select 改写成 if 条件语句。

##### 非阻塞操作

当 `select` 中包含两个 case 且其中一个是 default 的情况下，编译器会认为这是一次非阻塞收发操作。

发送情况：会使用条件语句和 `runtime.selectnbsend` 函数改写代码

接收情况：根据返回值数量的不同，会被改写成 `runtime.selectnbrecv` 和 `runtime.selctnbrecv2` 函数。

##### 常用流程

编译器使用如下流程处理多分支的 `select` 语句

* 将所有 case 转换成包含 channel 以及类型等信息的 `runtime.scase` 结构体
* 调用运行时 `runtime.selectgo` 从多个准备就绪的 channel 中选择一个可执行的 `runtime.scase` 结构体
* 通过 for 循环生成一组 if 语句，在语句中判断自己是不是被选中的 case。

随机顺序可以避免 channel 的饥饿问题，保证公平性。根据 channel 的地址顺序进行加锁能够避免死锁。

在 `runtime.selectgo` 中有三个阶段：

* 查找所有 case 中是否有可以立刻被处理的 channel
* 按需将 channel 加入到 sendq 和 recvq 队列中
* 最后从 `runtime.sudog` 中读取数据



### defer

