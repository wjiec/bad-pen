了解Kubernetes机理
-----------------------------

从这里开始我们研究Pod是如何被调度的，以及控制器管理器中的各种控制器是如何让部署的资源运行起来



### 了解架构

Kubernetes集群主要分为**控制平面**和**工作节点**。

其中控制平面负责控制并使得整个集群正常运转，控制平面上运行着**ApiServer、etcd分布式存储、Scheduler调度器、Controller-Manager控制器管理器**这些组件用来存储、管理集群状态。

而工作节点主要用于运行应用程序，其上有**kubelet、kube-proxy、容器运行时**等组件。

集群中还有一些附加组件如**kube-dns（提供服务的解析服务）、Ingress控制器（提供http(s)网关、路由等功能）、Dashboard仪表盘、容器网络接口插件**等组件用于增强Kubernetes的能力。

#### Kubernetes组件之间如何通信

* Kubernetes系统组件间智能通过ApiServer进行通信，组件之间不会直接通信
* ApiServer是唯一与etcd通信的组件，其他组件不会直接和etcd通信，而是通过ApiServer来修改集群状态
* ApiServer与其他组件的连接基本都是由组件发起的
  * **当使用`kubectl attach/exec/logs/port-forward`命令时，ApiServer会主动向kubelet发起请求**

#### Kubernetes组件的分布式特性

我们可以使用以下命令来检查每个控制平面组件的健康状态

```bash
kubectl get componentstatuses
```

为了保证高可用，**控制平面的每个组件都可以有多个实例**。etcd和api-server可以多个实例共同工作，而**调度器和控制器管理器在同一时间只能有一个实例起作用而其他实例处于待命状态**（主从模式）

**控制平面的组件以及kube-proxy可以直接部署在系统上或者作为Pod来运行**。而**kubelet是唯一一个需要做为常规系统组件部署在系统上运行的组件，它把其他组件作为Pod运行在Kubernetes中**。

#### Kubernetes如何使用etcd

etcd是一个响应快、分布式、一致的kv数据库。Kubernetes使用etcd作为持久化存储Pod、ReplicationController、Service和Secret等数据的手段。**唯一能直接和etcd通信的是Kubernetes的ApiServer**，所有其他组件都是通过与ApiServer间接读取、写入数据到etcd的。而且**etcd是Kubernetes存储集群状态和元数据的唯一地方**。

Kubernetes将所有数据存储（每个资源的完整JSON形式）到etcd的registry下（格式为`/registry/<resource>`），其中Pod是按照命名空间进行存储的（例如`/registry/pods/<namespace>/<pod-name>`），命名空间之下的每个条目对应一个单独的Pod。

#### ApiServer都做了什么

Kubernetes的ApiServer作为中心组件，其他组件或者客户端（如`kubectl`）都会去调用它。ApiServer以RESTful API的形式提供可以查询、修改集群状态的CURD接口，并将数据持久化到etcd中。

当一个请求发往ApiServer时，其内部逐次发生了以下流程（每个流程都会有多个插件共同参与本次请求的验证工作）：

* 通过认证插件认证客户端：**客户端的凭据是否是有效且经过认证的**
  * ApiServer会根据配置的一个或多个认证插件轮流来检查请求，直到有一个插件能确认是谁发送了请求为止
* 通过授权插件检查客户端：**客户端是否可以对所请求的资源执行操作**
  * ApiServer会根据配置的一个或多个授权插件来检查这个发出这个请求的用户是否有权限执行操作
* 通过准入插件验证资源请求：**客户端是否有权限去创建、修改或删除一个资源**
  * 如果请求只是尝试读取数据，则不会做准入控制的验证
  * 如果请求尝试创建、修改或删除一个资源，则ApiServer会根据配置的一个或多个准入插件来检查、修改甚至重写请求（比如添加默认值等）
* **将数据存储到etcd中，然后返回一个响应给到客户端**