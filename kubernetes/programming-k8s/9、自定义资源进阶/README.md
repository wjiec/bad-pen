自定义资源进阶
---

版本（Versioning）、转换（Conversion）和准入控制器（Adminission Controller）



### 自定义资源版本

多版本 API 时一个强大的功能，它可以在保持对旧版本客户端兼容的前提下对 API 进行改进。需要注意的是：版本转换功能要求 OpenAPI v3 的验证 Schema 是结构化的。



### 准入 Webhook

准入 Webhook 包括变更 Webhook 和验证准入 Webhook，而且它们的调用逻辑与原生资源也是相同的。
