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

>Go 编译器支持逃逸分析（escape analysis），它会在编译器通过构建调用图来分析局部变量是否会被外部引用，从而决定将变量分配到栈上还是堆上。
>
>编译参数 `-gcflags="-m"` 可以输出编译优化信息，其中包括内联和逃逸分析。



#### newobject的实现

```go
//
// runtime/malloc.go
//

// implementation of new builtin
// compiler (both frontend and SSA backend) knows the signature
// of this function
func newobject(typ *_type) unsafe.Pointer {
	return mallocgc(typ.size, typ, true)
}

// Allocate an object of size bytes.
// Small objects are allocated from the per-P cache's free lists.
// Large objects (> 32 kB) are allocated straight from the heap.
func mallocgc(size uintptr, typ *_type, needzero bool) unsafe.Pointer {
	if size == 0 {
		return unsafe.Pointer(&zerobase)
	}

	// assistG is the G to charge for this allocation, or nil if
	// GC is not currently active.
	var assistG *g
	if gcBlackenEnabled != 0 {
		// Charge the current user G for this allocation.
		assistG = getg()
		if assistG.m.curg != nil {
			assistG = assistG.m.curg
		}
		// Charge the allocation against the G. We'll account
		// for internal fragmentation at the end of mallocgc.
		assistG.gcAssistBytes -= int64(size)

		if assistG.gcAssistBytes < 0 {
			// This G is in debt. Assist the GC to correct
			// this before allocating. This must happen
			// before disabling preemption.
			gcAssistAlloc(assistG)
		}
	}

	// Set mp.mallocing to keep from being preempted by GC.
	mp := acquirem()
	if mp.mallocing != 0 {
		throw("malloc deadlock")
	}
	if mp.gsignal == getg() {
		throw("malloc during signal")
	}
	mp.mallocing = 1

	shouldhelpgc := false
	dataSize := size
    
    // 获取 mcache
	c := getMCache()
	if c == nil {
		throw("mallocgc called without a P or outside bootstrapping")
	}
	var span *mspan
	var x unsafe.Pointer
	noscan := typ == nil || typ.ptrdata == 0
	// In some cases block zeroing can profitably (for latency reduction purposes)
	// be delayed till preemption is possible; isZeroed tracks that state.
	isZeroed := true
    
    // 小对象分配
	if size <= maxSmallSize {
        // 非指针的小对象（<16字节）
		if noscan && size < maxTinySize {
			// Tiny allocator.
			//
			// Tiny allocator combines several tiny allocation requests
			// into a single memory block. The resulting memory block
			// is freed when all subobjects are unreachable. The subobjects
			// must be noscan (don't have pointers), this ensures that
			// the amount of potentially wasted memory is bounded.
			//
			// Size of the memory block used for combining (maxTinySize) is tunable.
			// Current setting is 16 bytes, which relates to 2x worst case memory
			// wastage (when all but one subobjects are unreachable).
			// 8 bytes would result in no wastage at all, but provides less
			// opportunities for combining.
			// 32 bytes provides more opportunities for combining,
			// but can lead to 4x worst case wastage.
			// The best case winning is 8x regardless of block size.
			//
			// Objects obtained from tiny allocator must not be freed explicitly.
			// So when an object will be freed explicitly, we ensure that
			// its size >= maxTinySize.
			//
			// SetFinalizer has a special case for objects potentially coming
			// from tiny allocator, it such case it allows to set finalizers
			// for an inner byte of a memory block.
			//
			// The main targets of tiny allocator are small strings and
			// standalone escaping variables. On a json benchmark
			// the allocator reduces number of allocations by ~12% and
			// reduces heap size by ~20%.
			off := c.tinyoffset
			// Align tiny pointer for required (conservative) alignment.
			if size&7 == 0 {
				off = alignUp(off, 8)
			} else if sys.PtrSize == 4 && size == 12 {
				// Conservatively align 12-byte objects to 8 bytes on 32-bit
				// systems so that objects whose first field is a 64-bit
				// value is aligned to 8 bytes and does not cause a fault on
				// atomic access. See issue 37262.
				// TODO(mknyszek): Remove this workaround if/when issue 36606
				// is resolved.
				off = alignUp(off, 8)
			} else if size&3 == 0 {
				off = alignUp(off, 4)
			} else if size&1 == 0 {
				off = alignUp(off, 2)
			}
            
            // 合并对象存储
			if off+size <= maxTinySize && c.tiny != 0 {
				// The object fits into existing tiny block.
				x = unsafe.Pointer(c.tiny + off)
				c.tinyoffset = off + size
				c.tinyAllocs++
				mp.mallocing = 0
				releasem(mp)
				return x
			}
            
            // 获取新的小对象内存规格
			// Allocate a new maxTinySize block.
			span = c.alloc[tinySpanClass]
			v := nextFreeFast(span)
			if v == 0 {
				v, span, shouldhelpgc = c.nextFree(tinySpanClass)
			}
			x = unsafe.Pointer(v)
			(*[2]uint64)(x)[0] = 0
			(*[2]uint64)(x)[1] = 0
			// See if we need to replace the existing tiny block with the new one
			// based on amount of remaining free space.
			if !raceenabled && (size < c.tinyoffset || c.tiny == 0) {
				// Note: disabled when race detector is on, see comment near end of this function.
				c.tiny = uintptr(x)
				c.tinyoffset = size
			}
			size = maxTinySize
		} else {
            // 带指针的小对象分配
			var sizeclass uint8
			if size <= smallSizeMax-8 {
				sizeclass = size_to_class8[divRoundUp(size, smallSizeDiv)]
			} else {
				sizeclass = size_to_class128[divRoundUp(size-smallSizeMax, largeSizeDiv)]
			}
			size = uintptr(class_to_size[sizeclass])
			spc := makeSpanClass(sizeclass, noscan)
			span = c.alloc[spc]
			v := nextFreeFast(span)
			if v == 0 {
				v, span, shouldhelpgc = c.nextFree(spc)
			}
			x = unsafe.Pointer(v)
			if needzero && span.needzero != 0 {
				memclrNoHeapPointers(unsafe.Pointer(v), size)
			}
		}
	} else {
        // 大对象直接从堆上分配
		shouldhelpgc = true
		// For large allocations, keep track of zeroed state so that
		// bulk zeroing can be happen later in a preemptible context.
		span, isZeroed = c.allocLarge(size, needzero && !noscan, noscan)
		span.freeindex = 1
		span.allocCount = 1
		x = unsafe.Pointer(span.base())
		size = span.elemsize
	}

	var scanSize uintptr
	if !noscan {
		// If allocating a defer+arg block, now that we've picked a malloc size
		// large enough to hold everything, cut the "asked for" size down to
		// just the defer header, so that the GC bitmap will record the arg block
		// as containing nothing at all (as if it were unused space at the end of
		// a malloc block caused by size rounding).
		// The defer arg areas are scanned as part of scanstack.
		if typ == deferType {
			dataSize = unsafe.Sizeof(_defer{})
		}
		heapBitsSetType(uintptr(x), size, dataSize, typ)
		if dataSize > typ.size {
			// Array allocation. If there are any
			// pointers, GC has to scan to the last
			// element.
			if typ.ptrdata != 0 {
				scanSize = dataSize - typ.size + typ.ptrdata
			}
		} else {
			scanSize = typ.ptrdata
		}
		c.scanAlloc += scanSize
	}

	// Ensure that the stores above that initialize x to
	// type-safe memory and set the heap bits occur before
	// the caller can make x observable to the garbage
	// collector. Otherwise, on weakly ordered machines,
	// the garbage collector could follow a pointer to x,
	// but see uninitialized memory or stale heap bits.
	publicationBarrier()

	// Allocate black during GC.
	// All slots hold nil so no scanning is needed.
	// This may be racing with GC so do it atomically if there can be
	// a race marking the bit.
	if gcphase != _GCoff {
		gcmarknewobject(span, uintptr(x), size, scanSize)
	}

	if raceenabled {
		racemalloc(x, size)
	}

	if msanenabled {
		msanmalloc(x, size)
	}

	if rate := MemProfileRate; rate > 0 {
		// Note cache c only valid while m acquired; see #47302
		if rate != 1 && size < c.nextSample {
			c.nextSample -= size
		} else {
			profilealloc(mp, x, size)
		}
	}
	mp.mallocing = 0
	releasem(mp)

	// Pointerfree data can be zeroed late in a context where preemption can occur.
	// x will keep the memory alive.
	if !isZeroed && needzero {
		memclrNoHeapPointersChunked(size, x) // This is a possible preemption point: see #47302
	}

	if debug.malloc {
		if debug.allocfreetrace != 0 {
			tracealloc(x, size, typ)
		}

		if inittrace.active && inittrace.id == getg().goid {
			// Init functions are executed sequentially in a single goroutine.
			inittrace.bytes += uint64(size)
		}
	}

	if assistG != nil {
		// Account for internal fragmentation in the assist
		// debt now that we know it.
		assistG.gcAssistBytes -= int64(size - dataSize)
	}

	if shouldhelpgc {
		if t := (gcTrigger{kind: gcTriggerHeap}); t.test() {
			gcStart(t)
		}
	}

	if raceenabled && noscan && dataSize < maxTinySize {
		// Pad tinysize allocations so they are aligned with the end
		// of the tinyalloc region. This ensures that any arithmetic
		// that goes off the top end of the object will be detectable
		// by checkptr (issue 38872).
		// Note that we disable tinyalloc when raceenabled for this to work.
		// TODO: This padding is only performed when the race detector
		// is enabled. It would be nice to enable it if any package
		// was compiled with checkptr, but there's no easy way to
		// detect that (especially at compile time).
		// TODO: enable this padding for all allocations, not just
		// tinyalloc ones. It's tricky because of pointer maps.
		// Maybe just all noscan objects?
		x = add(x, size-dataSize)
	}

	return x
}
```



#### 大对象分配

```go
//
// runtime/mcache.go
//

// allocLarge allocates a span for a large object.
// The boolean result indicates whether the span is known-zeroed.
// If it did not need to be zeroed, it may not have been zeroed;
// but if it came directly from the OS, it is already zeroed.
func (c *mcache) allocLarge(size uintptr, needzero bool, noscan bool) (*mspan, bool) {
	if size+_PageSize < size {
		throw("out of memory")
	}
    
    // 计算所需的页数
	npages := size >> _PageShift
	if size&_PageMask != 0 {
		npages++
	}

	// Deduct credit for this span allocation and sweep if
	// necessary. mHeap_Alloc will also sweep npages, so this only
	// pays the debt down to npage pages.
	deductSweepCredit(npages*_PageSize, npages)

    // 从 mheap 中获取 mspan
	spc := makeSpanClass(0, noscan)
	s, isZeroed := mheap_.alloc(npages, spc, needzero)
	if s == nil {
		throw("out of memory")
	}
	stats := memstats.heapStats.acquire()
	atomic.Xadduintptr(&stats.largeAlloc, npages*pageSize)
	atomic.Xadduintptr(&stats.largeAllocCount, 1)
	memstats.heapStats.release()

	// Update gcController.heapLive and revise pacing if needed.
	atomic.Xadd64(&gcController.heapLive, int64(npages*pageSize))
	if trace.enabled {
		// Trace that a heap alloc occurred because gcController.heapLive changed.
		traceHeapAlloc()
	}
	if gcBlackenEnabled != 0 {
		gcController.revise()
	}

	// Put the large span in the mcentral swept list so that it's
	// visible to the background sweeper.
	mheap_.central[spc].mcentral.fullSwept(mheap_.sweepgen).push(s)
	s.limit = s.base() + size
	heapBitsForAddr(s.base()).initSpan(s)
	return s, isZeroed
}
```



#### sweepgen字段

在每次GC时都会累加这个计数值。在 mheap 中的 mspan 不会被垃圾回收期关注，但 mcentral 中的 span 却有可能正在被清理，所以当 mcache 从 mcentral 中获取 mspan 时，该字段的值非常重要。

```go
type mspan struct {
    // if sweepgen == h.sweepgen - 2, the span needs sweeping
    // if sweepgen == h.sweepgen - 1, the span is currently swept
    // if sweepgen == h.sweepgen, the span is wept and ready to use
    sweepgen uint32
}
```



#### heap.alloc

从 mheap 中获取 mspan 的算法核心是找到大小最合适的块。首先从页数相同的链表查找，如没有结果，再从页数更多的链表获取，直至超大块或申请新块。如果返回了更大的 mspan，为避免浪费，会将多余部分切出来重新返回 mheap 链表，同时还将尝试合并相邻的闲置 mspan 空间，避免碎片。



### 回收

内存回收的源头是垃圾清理操作。之所以说回收而非释放，是因为整个内存分配器的核心是内存复用，不再使用的内存会被放回到合适的位置，等下次分配时再次使用。只有当空闲内存资源过多时，才会考虑释放。

出于效率考虑，回收操作不会直接盯着单个对象，而是以 mspan 为基本单位。通过对比 bitmap 里的扫描标记，逐步将 object 收归原 mspan，最终上交 mcentral 或 mheap。



### 释放

在运行时入口函数 `runtime.main` 里，会专门启动一个监控任务 `sysmon`，它每隔一段时间就会检查 mheap 里的闲置内存块。如果闲置时间超过阈值，则释放其关联的物理内存（在 unix 系统下仅通过调用 `modvise` 来通知操作系统某段内存暂时不用了，建议内核收回对应的物理内存）。



### 其他

从运行时角度，整个进程内的对象可分为两类，一类是从 marena 区域分配的用户对象，另一种则是运行时自身运行和管理所需的对象（mheap、mcache 等）。这些对象的生命周期相对较长且长度相对固定。所以运行时专门设计了 fixalloc 固定分配器来为这些对象分配内存。
