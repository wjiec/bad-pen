通道
------

通道是 Go 实现 CSP（Communicating Sequential Process）并发模型的关键，鼓励用通信来实现数据共享。

> Don't communicate by sharing memory, share memory by communicateing.



### 创建

创建通道时，需要注意的点是否需要缓冲槽

```go
//
// runtime/chan.go
//

type hchan struct {
	qcount   uint           // total data in the queue
	dataqsiz uint           // size of the circular queue
	buf      unsafe.Pointer // points to an array of dataqsiz elements
	elemsize uint16
	closed   uint32
	elemtype *_type // element type
	sendx    uint   // send index
	recvx    uint   // receive index
	recvq    waitq  // list of recv waiters
	sendq    waitq  // list of send waiters

	// lock protects all fields in hchan, as well as several
	// fields in sudogs blocked on this channel.
	//
	// Do not change another G's status while holding this lock
	// (in particular, do not ready a G), as this can deadlock
	// with stack shrinking.
	lock mutex
}

func makechan(t *chantype, size int) *hchan {
	elem := t.elem

	// compiler checks this but be safe.
	if elem.size >= 1<<16 {
		throw("makechan: invalid channel element type")
	}
	if hchanSize%maxAlign != 0 || elem.align > maxAlign {
		throw("makechan: bad alignment")
	}

	mem, overflow := math.MulUintptr(elem.size, uintptr(size))
	if overflow || mem > maxAlloc-hchanSize || size < 0 {
		panic(plainError("makechan: size out of range"))
	}

	// Hchan does not contain pointers interesting for GC when elements stored in buf do not contain pointers.
	// buf points into the same allocation, elemtype is persistent.
	// SudoG's are referenced from their owning thread so they can't be collected.
	// TODO(dvyukov,rlh): Rethink when collector can move allocated objects.
	var c *hchan
	switch {
	case mem == 0:
		// Queue or element size is zero.
		c = (*hchan)(mallocgc(hchanSize, nil, true))
		// Race detector uses this location for synchronization.
		c.buf = c.raceaddr()
	case elem.ptrdata == 0:
		// Elements do not contain pointers.
		// Allocate hchan and buf in one call.
		c = (*hchan)(mallocgc(hchanSize+mem, nil, true))
		c.buf = add(unsafe.Pointer(c), hchanSize)
	default:
		// Elements contain pointers.
		c = new(hchan)
		c.buf = mallocgc(mem, elem, true)
	}

	c.elemsize = uint16(elem.size)
	c.elemtype = elem
	c.dataqsiz = uint(size)
	lockInit(&c.lock, lockRankHchan)

	if debugChan {
		print("makechan: chan=", c, "; elemsize=", elem.size, "; dataqsiz=", size, "\n")
	}
	return c
}
```



### 同步模式

同步模式的关键是找到匹配的接收或发送方，找到则直接拷贝数据；找不到就将自身打包后放入等待队列，由另一方复制数据并唤醒。同步模式下，通道的作用仅仅是维护发送和接收队列，数据复制与通道无关。



### 异步模式

异步模式围绕缓冲槽进行，当有空位时，发送者向槽中复制数据；有数据后，接受者从槽中获取数据。双方都有唤醒排队的另一方继续工作的责任。



### 关闭

关闭操作间所有排队者唤醒，并通过 chan.closed、g.param 参数告知由 close 发出。



### 选择

选择模式（select）是从多个 channel 里随机选出可用的那个，编译器会将相关语句翻译成具体的函数调用。对于如下代码：

```go
func main() {
  c1, c2 := make(chan int), make(chan int, 2)

  select {
  case c1 <- 1:
    println("c1")
  case <- c2:
    println("c2")
  default:
    println("default")
  }
}
```

我们可以反编译得到以下内容（经过删减）：

```asm
"".main STEXT size=503 args=0x0 locals=0xb8 funcid=0x0
	0x0000 00000 (main.go:3)	TEXT	"".main(SB), ABIInternal, $184-0
	0x0000 00000 (main.go:3)	MOVQ	(TLS), CX
	0x0009 00009 (main.go:3)	LEAQ	-56(SP), AX
	0x000e 00014 (main.go:3)	CMPQ	AX, 16(CX)

	0x0012 00018 (main.go:3)	JLS	493
	0x0018 00024 (main.go:3)	SUBQ	$184, SP
	0x001f 00031 (main.go:3)	MOVQ	BP, 176(SP)
	0x0027 00039 (main.go:3)	LEAQ	176(SP), BP

	0x0043 00067 (main.go:4)	CALL	runtime.makechan(SB) // c1
	0x0069 00105 (main.go:4)	CALL	runtime.makechan(SB) // c2
	0x0134 00308 (main.go:6)	CALL	runtime.selectgo(SB) // select

	0x01ed 00493 (main.go:9)	NOP
	0x01ed 00493 (main.go:3)	CALL	runtime.morestack_noctxt(SB)
	0x01f2 00498 (main.go:3)	JMP	0
```

简化后的 select 流程如下：

* 用 pollorder “随机”遍历，找出准备好的 case
* 如果没有可用的 case，则尝试 default case
* **如都不可用，则将 selectG 打包放入所有 channel 的排队列表**
* 直到 selectG 被某个 channel 唤醒，遍历 ncase 并查找目标 case