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