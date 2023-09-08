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

#### 通过命令行使用 API

我们可以在终端中使用 `kubectl proxy --port=8080` 就可以把 Kubernetes API 服务器代理到本地，并处理了有关身份认证和授权相关的逻辑。接下来我们就可以直接使用 HTTP 来发送请求：

```shell
curl http://localhost:8080/apis/batch/v1
curl http://localhost:8080/api/v1/namespaces/kube-system/pods
```

##### API 服务器是如何处理请求的

当 Kubernetes API 服务器接收到一个 HTTP 请求后，HTTP 请求会经过一系列的过滤器（身份认证、变更准入、对象验证、验证准入、持久化）。