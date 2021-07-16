关于二分搜索算法的边界处理的一些思考
-------------------------------------------------------

二分搜索拥有非常好的时间复杂度，但是在使用二分搜索算法的过程中关于边界条件总是需要死记硬背还容易出错。

这里记录一下关于在二分搜索边界问题上我的一些思考。



### 遵循左闭右开原则

首先来吃一发Dijkstra的[安利][1]，总的来说就是左闭右开符合直觉，又可以省去代码中大量的`+1`和`-1`的边界条件检查。

这里我们就根据该原则有了以下代码

```python
def binary_search(array, target):
    left, right = 0, len(array)
    while left < right:
        mid = left + (right - left) // 2
        if array[mid] == target:
            return mid
        else if array[mid] < target:
            left = ?
        else:
            right = ?
    return left
```

接下来的重点就是如何赋值`left`和`right`



### 重新定义边界的值

既然是“原则”，那肯定需要从头到尾进行贯彻。首先我们在问号处已经有了一个结论：`array[mid] != target`即在索引`mid`不是我们需要查找的值。那根据**`左闭右开原则`**我们可以有以下结论：

* 左闭：左边**需要包含在下一次循环的边界中**且`mid`索引处不是我们需要找的值：`left = mid + 1`
* 右开：右边**不需要包含下一次循环的边界中**且`mid`索引处不是我们需要找的值：`right = mid`

由此就可以得出二分搜索的边界处理范式：

```python
def binary_search(array, target):
    left, right = 0, len(array)
    while left < right:
        mid = left + (right - left) // 2
        if array[mid] == target:
            return mid
        else if array[mid] < target:
            left = mid + 1
        else:
            right = mid
    return left
```



### 参考资料与内容

* [二分查找有几种写法？它们的区别是什么？][2]



[1]: https://www.cs.utexas.edu/users/EWD/transcriptions/EWD08xx/EWD831.html
[2]: https://www.zhihu.com/question/36132386
