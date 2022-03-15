开发应用的最佳实践
----------------------------

将Kubernetes上的资源与一个应用结合起来从全局的视角看



### Kubernetes中的资源

一个典型的manifest应用包含以下内容

* 通常应用由一个或者多个`Deployment`和`StatefulSet`对象进行管理
  * `Development`或`StatefulSet`是通过控制多个`ReplicaSet`进行滚动升级
    * `ReplicaSet`通过标签努力将系统的实际状态向期望状态靠拢
      * `ReplicaSet`只关心Pod的数量是否匹配期望的副本数量（不管Pod处于什么状态）

* 其中包含了一个或者多个容器的模板（`Template`）
  * 每个容器都有一个存活探针（`livenessProbe`）和一个就绪探针（`readinessProbe`）
* 提供服务的Pod通过一个或者多个Service来暴露自己
  * 当需要提供外部访问时配置为`LoadBanancer`或`NodePort`类型的服务
  * 而`StatefulSet`类型的应用或者特殊情况下可能会用到`HeadlessService(ClusterIP: none)`
* 如果提供的是HTTP相关的服务还可以通过`Ingress`来开放指定路由主机名的服务
* `Pod`通常会引用两种类型的`Secret`
  * 一种是用来从私有仓库拉取镜像的`ImagePullSecret`
  * 另一种是应用程序运行时所需要的，一般由运维人员配置（如数据库账号密码等）
    * 这类`Secret`通常会被分配到`ServiceAccount`，然后再由`ServiceAccount`分配给单独的`Pod`
* `Pod`一般还包含一个或者多个的`ConfigMap`对象，我们可以用它来初始化环境变量或者以卷的方式进行挂载
* 有一些Pod可能会使用额外的卷用来在容器间共享文件或是进行持久化存储
  * 使用`emptyDir`卷来在Pod的多个容器将共享文件
  * 使用`PersistentVolumeClaim(PVC)`来持久化应用程序数据
    * `PersistentVolumeClaim(PVC)`会引用一个`PersistentVolume(PV)`
      * `PV`被一个`PVC`绑定后有不同的回收方式
      * `PV`可以被手工创建或者通过`StorageClass`生成
        * `StorageClass`由运维人员配置，可以根据需求自动创建相对应的`PV`
* 某些情况下，一个应用可能还需要使用一次性任务`Job`或者定时任务`CronJob`来处理某些任务
* 而这些应用可能会依赖一些由`DaemonSet`创建的系统级Pod
* 同时集群管理员还会为Pod创建`LimitRange`（Pod级别资源控制）或者`ResourceQuota`（命名空间级别资源控制）以控制计算资源使用



### 了解Pod的生命周期

Pod中运行的应用程序随时有可能被杀死，因为Kubernetes可能需要将这个Pod独爱度到另外一个节点或是需要进行应用缩容。

#### 应用必须预料到会被杀死或重新调度

当应用运行在Kubernetes中，这意味着应用可以更加频繁地进行自动迁移而无需人工干预，这需要应用开发者必须允许应用程序随时可被杀死而不会造成其他影响。

##### 预料到IP和主机名会发生变化

当一个Pod被杀死并且在其他地方运行之后，他不仅拥有了一个新的IP地址还有一个新的主机名。这需要应用程序不依赖成员的IP地址或者主机名来构建彼此的关系。**如果需要使用主机名来构建关系，则必须使用`StatefulSet`**。

##### 预料到写入磁盘的数据会消失

当应用被杀死并在一个新Pod中启动时，原有容器中写入的数据会丢失，除非将这些数据持久化到一个持久卷中，否则当Pod被重新调度时，丢失数据是一定的。

##### 使用存储卷来跨容器持久化数据

为了保证当Pod重启时数据不丢失，我们需要使用Pod级别的持久卷（持久卷的生命周期与Pod绑定），这样新容器就可以重用之前容器写到卷上的数据（PVC的名字相同则会引用相同的PV）。**但这可能带来的问题是如果前一个Pod的应用程序在持久卷中写入了错误的数据可能会导致新Pod无法启动并进入`CrashLoopBackOff`（循环崩溃）状态**。

##### 重新调度死亡或部分死亡的Pod

当我们使用`ReplicaSet`或`ReplicationController`等副本控制器托管Pod时，这些副本控制器并不关系Pod的实际状态，而**只在意Pod的数量是否与期望副本数匹配**。这意味着当Pod失败而死亡时，Kubneretes不会重新将其进行重新调度。

#### 以固定的顺序启动Pod

