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

除此之外，ApiServer不会做其他额外的操作。

#### ApiServer如何融资客户端资源变更

创建Pod，管理服务（Service）的端点（Endpoint）这是控制器管理器的工作。而这些操作是控制平面组件通过向ApiServer订阅资源的变更（创建、修改、删除）通知来实现的。

客户端通过创建到ApiServer的HTTP连接来监听资源的变更，每当资源更新时，服务器都会把新版本对象发送至所有监听该对象的客户端。

> 当客户端创建一个Pod时（发送YAML或JSON到ApiServer），ApiServer经过鉴权后会将完整的配置存储到etcd中，然后通知所有监听该Pod资源的客户端。

#### 了解调度器

调度器的工作简单说就是利用ApiServer的监听机制等待新创建的Pod，然后给每个新的、没有节点分配数据的Pod分配节点（增加节点数据）。**需要注意的是，调度器不会直接命令选中的节点（或是通知节点上的kubelet）去运行Pod**。

调度器做的只是通过ApiServer更新Pod的定义，然后由ApiServer再去通知kubelet（订阅了Pod资源的变更），当节点上的kubelet发现该Pod是调度到本节点的，就会去创建并运行Pod的容器。

实际上调度器如何为Pod选择一个最佳节点是一个比较困难的操作，选择节点操作可以分解为两部分

* 过滤所有节点，找出能分配给Pod的可用节点列表
* 对可用节点按优先级排序，找出最优节点。如果有多个节点都有相同的最高优先级，那么则循环分配，确保拥有相对平均的Pod

加入一个Pod有多个副本，理想情况下，我们会期望副本能够分散在尽可能多的节点上，而不是全部分配到单独的一个节点，这样有助于提高容错率和可靠性。调度器可以配置成满足特定需求或者基础设施特性，也可以整体替换为一个定制的实现。

在一个集群中可以运行多个调度器而非单个，对于每一个Pod，都可以通过在Pod中配置`schedulerName`来指定使用特定的调度器来调度特定的Pod。