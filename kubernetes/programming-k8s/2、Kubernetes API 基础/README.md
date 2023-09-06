Kubernetes API 基础
---

Kubernetes 集群由一系列节点组成，这些节点分属于不同的角色。API 服务器是完成中心管理的实体，也是唯一一个直接与分布式存储组件 etcd 交互的组件。



### API 服务器

从客户端角度来看，API 服务器提供了一个 RESTful HTTP API，提供 JSON 或者 Protobuf 格式的数据内容。

#### API 术语

* Kind：一个实体的类型，对应一个 Golang 类型
* API Group：一组逻辑上相关的 Kind
* Version：每个 API Group 的组，多个版本允许共存

#### Kubernetes API 版本

Kibernetes 的 API 分为核心组（`/api/v1`）和其他有名字的组（`/apis/$name/$version`）

#### 申明式状态管理

大部分 API 对象都区分资源的期望状态和当前状态。spec 是对于某种资源的期望状态的完整描述，通常会持久化到存储系统（比如 etcd）中。