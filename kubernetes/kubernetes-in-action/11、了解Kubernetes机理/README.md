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

#### ApiServer如何通知客户端资源变更

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

#### 控制器管理器中运行的控制器

控制器管理器负责确保系统的真实状态向ApiServer定义的期望状态收敛。资源描述了集群中应该运行什么，而控制器会去做具体的工作来部署资源。控制器都是通过ApiServer监听资源变更，并且不论是创建、更新或删除已有对象，控制器都会会变更执行相应的操作。控制器之间不会直接通信，它们甚至不知道其他控制器的存在。

总的来说，控制器执行一个“调度”循环，将实际状态调整为期望状态，然后将新的实际状态写入资源的`staus`部分。由于监听机制并不一定保证不漏掉消息，所以控制器还会定期执行查询操作来确保不会丢掉什么。

常见的控制器有以下几种：

* Replication管理器（ReplicationController资源的管理器）：监听可能影响期望状态耳朵复制集（replica）的数量和符合条件的Pod的数量变更事件，并在相应事件发生时做出相应的动作。需要注意的是Replication管理器不会去运行Pod，而是向ApiServer提交新的Pod资源。
* ReplicaSet、DaemonSet以及Job控制器：与Replication管理器类似，都会从各自的Pod模板中创建相应的资源并提交到ApiServer。
* Deployment控制器：每次Deployment对象修改后（会影响到部署的Pod），控制器都会滚动升级到新的版本。
* StatefulSet控制器：类似于Replication管理器，同时会初始化并管理每个Pod实例的持久卷声明（PVC）字段。
* Node控制器：管理Node资源，并监控每个节点的健康状态，删除不可达节点的Pod。
* Service控制器：负责从基础设施里请求一个负载均衡器使得LoadBalancer服务可用，并在删除服务时从基础设施里释放负载均衡器。
* Endpoint控制器：当Service或者相关联的Pod更新时，创建或删除对应的端点列表。
* Namespace控制器：当命名空间被删除时，删除属于该命名空间的所有资源。
* PersistentVolume控制器：为持久卷声明（PVC）找到最合适的持久卷并将其绑定。

#### kubelet做了什么

kubelet是负责管理所有运行在工作节点上的内容组件。他一般都按照顺序会有以下几个任务

* 启动时在ApiServer中通过创建一个Node资源来注册该节点
* 持续监控ApiServer是否有将Pod分配这个节点，并运行容器
* 随后持续监控运行的容器并向ApiServer报告它们的状态、事件和资源消耗
* 同时也是运行容器存活、就绪探针，当探针发生失败时重启容器
* 当Pod从ApiServer删除时，kubelet终止容器并通知ApiServer

kubelet一般会通过ApiServer来获取Pod列表，它也可以基于本地指定目录下的Pod清单来运行Pod。

#### kube-proxy的作用

kube-proxy用于确保客户端可以通过Kubernetes API连接到你定义的服务，同时确保对服务IP和端口的连接最终能达到某个Pod上，如果有多个Pod支持一个服务，那么代理会起到负载均衡的作用。

kube-proxy有几种代理模式：

* userspace模式：kube-proxy的最初实现方式，通过配置iptables将流量转发给kube-proxy服务，然后kube-proxy再将流量转发某个具体的Pod（轮询）。
* iptables模式：通过配置iptables规则让内核将流量转发给随机的一个Pod
* ipvs模式：通过配置ipvs虚拟IP的方式来将流量随机转发到某个Pod

#### Kubernetes插件

通过将YAML文件提交给ApiServer，这些组件会作为Pod部署在集群中并以插件的形式进行工作。这些组件是通过Deployment资源或者ReplicationController资源或者也可以是DaemonSet资源来部署在集群中的。

#### DNS服务器如何工作

集群中所有Pod默认都会使用哈集群内部的DNS服务，这使得Pod能够轻松地通过名称查询到服务，甚至是屋头服务的Pod地址。DNS服务通过在每个Pod中的`/etc/resolve.conf`中的`nameserver`来定义。

DNS服务利用ApiServer的监听机制来订阅Service和Endpoint的变动使得客户端总是能获取到最新的DNS信息。

#### Ingress控制器如何工作

Ingress控制器运行一个反向代理服务器，并根据集群中定义的Ingress、Service以及Endpint资源来配置其中的反向代理程序。

需要注意的是虽然Ingress资源定义指向一个Service，但是Ingress控制器会直接将流量转发给对应的Pod而不经过服务（通过查询Endpoint实现）



### 控制器如何协作

