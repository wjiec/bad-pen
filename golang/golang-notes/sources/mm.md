内存管理
-------------

内置运行时的编程语言通常会抛弃传统的内存分配方式，该由自主管理。以下是一些内存分配算法的基本策略：

* 每次从操作系统申请一大块内存，以减少系统调用
* 将申请到的大块内存按照特定大小预先切分成小块，构成链表
* 为对象分配内存时，只需从大小合适的链表中取一个即可
* 回收对象内存时，将该小块内存重新归还到原链表，以便复用
* 如闲置内存过多，则尝试归还部分内存给操作系统，降低整体开销



内存分配器只管理内存，并不关心对象状态。且它不会主动回收内存，垃圾回收期在完成清理操作后，触发内存分配器的回收操作。内存分配器将其管理的内存分为两种：

* span：由多个地址连续的页（page）组成的大块内存
* object：将 span 按照特定大小切分成多个大小相等的小块，每个小块可存储一个对象

内存分配器按页数来区分不同大小的 span。同时，span 的大小并非固定不变，在获取闲置 span 时，如果没找到大小合适的额，那就返回页数更多的，此时会引发裁剪操作，多余部分将构成新的 span 被放回到 heap 中。内存分配器还会尝试将地址相邻的空闲 span 合并，以构建更大的内存块，减少碎片，同时提供更灵活的分配策略。

```go
//
// runtime/malloc.go
//
const (
    _PageShift      = 13
	_PageSize = 1 << _PageShift // 8KB
)
```

用于存储对象的 object，按 8 字节倍数分为 `_NumSizeClasses = 68` 种，比如大小为 24 字节的 object 可用来存储 17 - 24 字节的对象。虽然会造成一定的内存浪费，但分配器只需要管理有限的几种规格，有助于分配和复用管理策略。

```go
//
// runtime/sizeclass.go
//
const (
	_NumSizeClasses = 68
)
```

内存分配器在分配内存时会根据内存块大小和规格的表格来选择合适的大小。若对象大小超出特定的阈值，则会被当成大对象（large object）直接在 mheap 上进行分配（此时返回的 sizeclass = 0）

#### 内存分配器中的组件

内存分配器的几个组件的结构定义如下：

* mcache：每个线程都会绑定到一个 mcache 上，用于无锁的 object 分配
* mcentral：为所有的 mcache 提供切分好的后备 span 资源，在 mcache 无法分配指定大小的对象时，尝试从 mcentral 中分配
* mheap：管理所有闲置的 mspan，需要时向操作系统申请新内存

```go
//
// runtime/mheap.go
//

// Main malloc heap.
// The heap itself is the "free" and "scav" treaps,
// but all the other global data is here too.
//
// mheap must not be heap-allocated because it contains mSpanLists,
// which must not be heap-allocated.
//
//go:notinheap
type mheap struct {
	// lock must only be acquired on the system stack, otherwise a g
	// could self-deadlock if its stack grows with the lock held.
	lock  mutex
	pages pageAlloc // page allocation data structure

	sweepgen     uint32 // sweep generation, see comment in mspan; written during STW
	sweepDrained uint32 // all spans are swept or are being swept
	sweepers     uint32 // number of active sweepone calls

	// allspans is a slice of all mspans ever created. Each mspan
	// appears exactly once.
	//
	// The memory for allspans is manually managed and can be
	// reallocated and move as the heap grows.
	//
	// In general, allspans is protected by mheap_.lock, which
	// prevents concurrent access as well as freeing the backing
	// store. Accesses during STW might not hold the lock, but
	// must ensure that allocation cannot happen around the
	// access (since that may free the backing store).
	allspans []*mspan // all spans out there

	_ uint32 // align uint64 fields on 32-bit for atomics

	// Proportional sweep
	//
	// These parameters represent a linear function from gcController.heapLive
	// to page sweep count. The proportional sweep system works to
	// stay in the black by keeping the current page sweep count
	// above this line at the current gcController.heapLive.
	//
	// The line has slope sweepPagesPerByte and passes through a
	// basis point at (sweepHeapLiveBasis, pagesSweptBasis). At
	// any given time, the system is at (gcController.heapLive,
	// pagesSwept) in this space.
	//
	// It's important that the line pass through a point we
	// control rather than simply starting at a (0,0) origin
	// because that lets us adjust sweep pacing at any time while
	// accounting for current progress. If we could only adjust
	// the slope, it would create a discontinuity in debt if any
	// progress has already been made.
	pagesInUse         uint64  // pages of spans in stats mSpanInUse; updated atomically
	pagesSwept         uint64  // pages swept this cycle; updated atomically
	pagesSweptBasis    uint64  // pagesSwept to use as the origin of the sweep ratio; updated atomically
	sweepHeapLiveBasis uint64  // value of gcController.heapLive to use as the origin of sweep ratio; written with lock, read without
	sweepPagesPerByte  float64 // proportional sweep ratio; written with lock, read without
	// TODO(austin): pagesInUse should be a uintptr, but the 386
	// compiler can't 8-byte align fields.

	// scavengeGoal is the amount of total retained heap memory (measured by
	// heapRetained) that the runtime will try to maintain by returning memory
	// to the OS.
	scavengeGoal uint64

	// Page reclaimer state

	// reclaimIndex is the page index in allArenas of next page to
	// reclaim. Specifically, it refers to page (i %
	// pagesPerArena) of arena allArenas[i / pagesPerArena].
	//
	// If this is >= 1<<63, the page reclaimer is done scanning
	// the page marks.
	//
	// This is accessed atomically.
	reclaimIndex uint64
	// reclaimCredit is spare credit for extra pages swept. Since
	// the page reclaimer works in large chunks, it may reclaim
	// more than requested. Any spare pages released go to this
	// credit pool.
	//
	// This is accessed atomically.
	reclaimCredit uintptr

	// arenas is the heap arena map. It points to the metadata for
	// the heap for every arena frame of the entire usable virtual
	// address space.
	//
	// Use arenaIndex to compute indexes into this array.
	//
	// For regions of the address space that are not backed by the
	// Go heap, the arena map contains nil.
	//
	// Modifications are protected by mheap_.lock. Reads can be
	// performed without locking; however, a given entry can
	// transition from nil to non-nil at any time when the lock
	// isn't held. (Entries never transitions back to nil.)
	//
	// In general, this is a two-level mapping consisting of an L1
	// map and possibly many L2 maps. This saves space when there
	// are a huge number of arena frames. However, on many
	// platforms (even 64-bit), arenaL1Bits is 0, making this
	// effectively a single-level map. In this case, arenas[0]
	// will never be nil.
	arenas [1 << arenaL1Bits]*[1 << arenaL2Bits]*heapArena

	// heapArenaAlloc is pre-reserved space for allocating heapArena
	// objects. This is only used on 32-bit, where we pre-reserve
	// this space to avoid interleaving it with the heap itself.
	heapArenaAlloc linearAlloc

	// arenaHints is a list of addresses at which to attempt to
	// add more heap arenas. This is initially populated with a
	// set of general hint addresses, and grown with the bounds of
	// actual heap arena ranges.
	arenaHints *arenaHint

	// arena is a pre-reserved space for allocating heap arenas
	// (the actual arenas). This is only used on 32-bit.
	arena linearAlloc

	// allArenas is the arenaIndex of every mapped arena. This can
	// be used to iterate through the address space.
	//
	// Access is protected by mheap_.lock. However, since this is
	// append-only and old backing arrays are never freed, it is
	// safe to acquire mheap_.lock, copy the slice header, and
	// then release mheap_.lock.
	allArenas []arenaIdx

	// sweepArenas is a snapshot of allArenas taken at the
	// beginning of the sweep cycle. This can be read safely by
	// simply blocking GC (by disabling preemption).
	sweepArenas []arenaIdx

	// markArenas is a snapshot of allArenas taken at the beginning
	// of the mark cycle. Because allArenas is append-only, neither
	// this slice nor its contents will change during the mark, so
	// it can be read safely.
	markArenas []arenaIdx

	// curArena is the arena that the heap is currently growing
	// into. This should always be physPageSize-aligned.
	curArena struct {
		base, end uintptr
	}

	_ uint32 // ensure 64-bit alignment of central

	// central free lists for small size classes.
	// the padding makes sure that the mcentrals are
	// spaced CacheLinePadSize bytes apart, so that each mcentral.lock
	// gets its own cache line.
	// central is indexed by spanClass.
	central [numSpanClasses]struct {
		mcentral mcentral
		pad      [cpu.CacheLinePadSize - unsafe.Sizeof(mcentral{})%cpu.CacheLinePadSize]byte
	}

	spanalloc             fixalloc // allocator for span*
	cachealloc            fixalloc // allocator for mcache*
	specialfinalizeralloc fixalloc // allocator for specialfinalizer*
	specialprofilealloc   fixalloc // allocator for specialprofile*
	specialReachableAlloc fixalloc // allocator for specialReachable
	speciallock           mutex    // lock for special record allocators.
	arenaHintAlloc        fixalloc // allocator for arenaHints

	unused *specialfinalizer // never set, just here to force the specialfinalizer type into DWARF
}


// Central list of free objects of a given size.
//
//go:notinheap
type mcentral struct {
	spanclass spanClass

	// partial and full contain two mspan sets: one of swept in-use
	// spans, and one of unswept in-use spans. These two trade
	// roles on each GC cycle. The unswept set is drained either by
	// allocation or by the background sweeper in every GC cycle,
	// so only two roles are necessary.
	//
	// sweepgen is increased by 2 on each GC cycle, so the swept
	// spans are in partial[sweepgen/2%2] and the unswept spans are in
	// partial[1-sweepgen/2%2]. Sweeping pops spans from the
	// unswept set and pushes spans that are still in-use on the
	// swept set. Likewise, allocating an in-use span pushes it
	// on the swept set.
	//
	// Some parts of the sweeper can sweep arbitrary spans, and hence
	// can't remove them from the unswept set, but will add the span
	// to the appropriate swept list. As a result, the parts of the
	// sweeper and mcentral that do consume from the unswept list may
	// encounter swept spans, and these should be ignored.
	partial [2]spanSet // list of spans with a free object
	full    [2]spanSet // list of spans with no free objects
}


// Per-thread (in Go, per-P) cache for small objects.
// This includes a small object cache and local allocation stats.
// No locking needed because it is per-thread (per-P).
//
// mcaches are allocated from non-GC'd memory, so any heap pointers
// must be specially handled.
//
//go:notinheap
type mcache struct {
	// The following members are accessed on every malloc,
	// so they are grouped here for better caching.
	nextSample uintptr // trigger heap sample after allocating this many bytes
	scanAlloc  uintptr // bytes of scannable heap allocated

	// Allocator cache for tiny objects w/o pointers.
	// See "Tiny allocator" comment in malloc.go.

	// tiny points to the beginning of the current tiny block, or
	// nil if there is no current tiny block.
	//
	// tiny is a heap pointer. Since mcache is in non-GC'd memory,
	// we handle it by clearing it in releaseAll during mark
	// termination.
	//
	// tinyAllocs is the number of tiny allocations performed
	// by the P that owns this mcache.
	tiny       uintptr
	tinyoffset uintptr
	tinyAllocs uintptr

	// The rest is not accessed on every malloc.

	alloc [numSpanClasses]*mspan // spans to allocate from, indexed by spanClass

	stackcache [_NumStackOrders]stackfreelist

	// flushGen indicates the sweepgen during which this mcache
	// was last flushed. If flushGen != mheap_.sweepgen, the spans
	// in this mcache are stale and need to the flushed so they
	// can be swept. This is done in acquirep.
	flushGen uint32
}


//go:notinheap
type mspan struct {
	next *mspan     // next span in list, or nil if none
	prev *mspan     // previous span in list, or nil if none
	list *mSpanList // For debugging. TODO: Remove.

	startAddr uintptr // address of first byte of span aka s.base()
	npages    uintptr // number of pages in span

	manualFreeList gclinkptr // list of free objects in mSpanManual spans

	// freeindex is the slot index between 0 and nelems at which to begin scanning
	// for the next free object in this span.
	// Each allocation scans allocBits starting at freeindex until it encounters a 0
	// indicating a free object. freeindex is then adjusted so that subsequent scans begin
	// just past the newly discovered free object.
	//
	// If freeindex == nelem, this span has no free objects.
	//
	// allocBits is a bitmap of objects in this span.
	// If n >= freeindex and allocBits[n/8] & (1<<(n%8)) is 0
	// then object n is free;
	// otherwise, object n is allocated. Bits starting at nelem are
	// undefined and should never be referenced.
	//
	// Object n starts at address n*elemsize + (start << pageShift).
	freeindex uintptr
	// TODO: Look up nelems from sizeclass and remove this field if it
	// helps performance.
	nelems uintptr // number of object in the span.

	// Cache of the allocBits at freeindex. allocCache is shifted
	// such that the lowest bit corresponds to the bit freeindex.
	// allocCache holds the complement of allocBits, thus allowing
	// ctz (count trailing zero) to use it directly.
	// allocCache may contain bits beyond s.nelems; the caller must ignore
	// these.
	allocCache uint64

	// allocBits and gcmarkBits hold pointers to a span's mark and
	// allocation bits. The pointers are 8 byte aligned.
	// There are three arenas where this data is held.
	// free: Dirty arenas that are no longer accessed
	//       and can be reused.
	// next: Holds information to be used in the next GC cycle.
	// current: Information being used during this GC cycle.
	// previous: Information being used during the last GC cycle.
	// A new GC cycle starts with the call to finishsweep_m.
	// finishsweep_m moves the previous arena to the free arena,
	// the current arena to the previous arena, and
	// the next arena to the current arena.
	// The next arena is populated as the spans request
	// memory to hold gcmarkBits for the next GC cycle as well
	// as allocBits for newly allocated spans.
	//
	// The pointer arithmetic is done "by hand" instead of using
	// arrays to avoid bounds checks along critical performance
	// paths.
	// The sweep will free the old allocBits and set allocBits to the
	// gcmarkBits. The gcmarkBits are replaced with a fresh zeroed
	// out memory.
	allocBits  *gcBits
	gcmarkBits *gcBits

	// sweep generation:
	// if sweepgen == h->sweepgen - 2, the span needs sweeping
	// if sweepgen == h->sweepgen - 1, the span is currently being swept
	// if sweepgen == h->sweepgen, the span is swept and ready to use
	// if sweepgen == h->sweepgen + 1, the span was cached before sweep began and is still cached, and needs sweeping
	// if sweepgen == h->sweepgen + 3, the span was swept and then cached and is still cached
	// h->sweepgen is incremented by 2 after every GC

	sweepgen    uint32
	divMul      uint32        // for divide by elemsize
	allocCount  uint16        // number of allocated objects
	spanclass   spanClass     // size class and noscan (uint8)
	state       mSpanStateBox // mSpanInUse etc; accessed atomically (get/set methods)
	needzero    uint8         // needs to be zeroed before allocation
	elemsize    uintptr       // computed from sizeclass or from npages
	limit       uintptr       // end of data in span
	speciallock mutex         // guards specials list
	specials    *special      // linked list of special records sorted by offset.
}
```



#### 内存分配及回收流程

分配流程：

* 计算待分配对象的内存规格（sizeclass）
* 从 `mcache.alloc` 数组中找到对应规格的 `mspan` （`alloc [numSpanClasses]*mspan`）
* 从 `mspan.manualFreeList` 链表中提取可用的 object（`manualFreeList gclinkptr // list of free objects in mSpanManual spans`）
  * 如果 `mspan.manualFreeList` 为空，则从 `mcentral.partial` 中获取新的 span
    * 如果 `mcentral.partial` 为空，则从 `mheap` 中获取，并切分成 object 链表
      * 如果 `mheap` 中也没有合适的闲置 span，则向操作系统申请新内存块

释放流程：

* 将标记为可回收的 object 交还给所属的 `mspan`
* 同时 `mspan` 可能被放回到 `mcentral` 中，供其他任意 `mcache` 重新获取使用
* 如果 `mspan` 已收回全部 object，则将其交还给 `mheap`，以便重新切分复用
* 定期扫描 `mheap` 里长时间闲置的 `mspan`，释放其占用的内存



#### 内存分配器高效的秘诀

线程私有且不被共享的 mcache 是实现高性能无锁分配的核心，而 mcentral 的作用是在多个 mcache 间提高 object 利用率，避免内存浪费。将 span 归还给 mheap 是为了平衡不同规格 object 的需求。



### 初始化（过时的）

内存分配器和垃圾回收算法都依赖于连续地址，所以在初始化阶段，预先保留了很大的一段虚拟地址空间（并不会分配内存）。这一段空间会会划分为三个区域：

* `spans`：512M，用于表示页所属 span 的指针数组
* `bitmap`：32GB，GC标记位图
* `arena`：512GB，用户内存分配区域

#### mmap

函数 mmap 要求操作系统内核创建新的虚拟存储器区域，可指定起始地址和长度。Windows 没有此函数，对应的 API 是 `VirtualAlloc`。操作系统在对用户的内存申请请求时大多采取机会主义分配策略，申请内存时，仅承诺但不是立即分配物理内存。物理内存的分配在写操作导致缺页异常时发生，通常是按页提供的。



### 分配（过时的）

为对象分配内存需区分是在栈上还是在堆上完成。通常情况下，编译器会尽可能地使用寄存器和栈来存储对象，这有助于提升性能，减少垃圾回收器压力。

对于同一段代码，在是否启用内联的情况下也会出现不同的情况：

```go
package main

func assign() *int {
  x := new(int)
  *x = 1234
  return x
}

func main() {
  println(*assign())
}
```

此时我们通过 `-gcflags="-l"` 禁用内联，并使用 `go tool objdump -s "assign" <binary>` 来查看汇编代码

```asm
//
// go build -gcflags="-l" assign.go
// go tool objdump -s "main\.assign" assign
//

TEXT main.assign(SB) /root/workspace/gonotes/mm/assign.go
  assign.go:3		0x4553e0		493b6610		CMPQ 0x10(R14), SP			
  assign.go:3		0x4553e4		7630			JBE 0x455416				
  assign.go:3		0x4553e6		4883ec18		SUBQ $0x18, SP				
  assign.go:3		0x4553ea		48896c2410		MOVQ BP, 0x10(SP)			
  assign.go:3		0x4553ef		488d6c2410		LEAQ 0x10(SP), BP			
  assign.go:4		0x4553f4		488d0545490000		LEAQ 0x4945(IP), AX			
  assign.go:4		0x4553fb		0f1f440000		NOPL 0(AX)(AX*1)			
  assign.go:4		0x455400		e8fb58fbff		CALL runtime.newobject(SB)		
  assign.go:5		0x455405		48c700d2040000		MOVQ $0x4d2, 0(AX)			
  assign.go:6		0x45540c		488b6c2410		MOVQ 0x10(SP), BP			
  assign.go:6		0x455411		4883c418		ADDQ $0x18, SP				
  assign.go:6		0x455415		c3			RET					
  assign.go:3		0x455416		e845ceffff		CALL runtime.morestack_noctxt.abi0(SB)	
  assign.go:3		0x45541b		ebc3			JMP main.assign(SB)
```

可以发现，此时编译器调用 `runtime.newobject` 在堆上分配内存。如果我们启用内联：

```asm
//
// go build assign.go
// go tool objdump -s "main\.main" assign
//

TEXT main.main(SB) /root/workspace/gonotes/mm/assign.go
  assign.go:9		0x4553e0		493b6610		CMPQ 0x10(R14), SP			
  assign.go:9		0x4553e4		7633			JBE 0x455419				
  assign.go:9		0x4553e6		4883ec10		SUBQ $0x10, SP				
  assign.go:9		0x4553ea		48896c2408		MOVQ BP, 0x8(SP)			
  assign.go:9		0x4553ef		488d6c2408		LEAQ 0x8(SP), BP			
  assign.go:10		0x4553f4		e8477cfdff		CALL runtime.printlock(SB)		
  assign.go:10		0x4553f9		b8d2040000		MOVL $0x4d2, AX				
  assign.go:10		0x4553fe		6690			NOPW					
  assign.go:10		0x455400		e85b83fdff		CALL runtime.printint(SB)		
  assign.go:10		0x455405		e8b67efdff		CALL runtime.printnl(SB)		
  assign.go:10		0x45540a		e8b17cfdff		CALL runtime.printunlock(SB)		
  assign.go:11		0x45540f		488b6c2408		MOVQ 0x8(SP), BP			
  assign.go:11		0x455414		4883c410		ADDQ $0x10, SP				
  assign.go:11		0x455418		c3			RET					
  assign.go:9		0x455419		e842ceffff		CALL runtime.morestack_noctxt.abi0(SB)	
  assign.go:9		0x45541e		6690			NOPW					
  assign.go:9		0x455420		ebbe			JMP main.main(SB)			
```

