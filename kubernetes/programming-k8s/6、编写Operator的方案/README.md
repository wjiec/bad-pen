编写 Operator 的方案
---

使用合适的方案来编写 Operator 可以避免很多重复的代码开发工作，让我们更专注于业务逻辑的开发，更快地开展工作，提高效率。



### 基于 sample-controller

我们可以基于 `k8s.io/sample-controller` 来实现自定义的 Operator。详细参考代码实现。