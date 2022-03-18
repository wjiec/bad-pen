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

Kubernetes与运维人员手动部署应用有一点不同，运维人员知道应用间的依赖关系这样他们就可以按照顺序来启动应用。而Kubernets没有内置的方法来先运行某些Pod然后等这些Pod运行成功后再运行其他Pod（Kubernetes的确是按照YAML文件中定义的顺序来处理的，但是这只保证它们被写到etcd的时候是有顺序的，而无法确保Pod会按这个顺序启动）。

**但是我们可以阻止一个主容器的启动，直到它的前置条件被满足。这是通过在Pod中包含一个叫做init的容器实现的。**

##### init容器使用

Pod中的init容器可以用来初始化Pod，比如往容器的存储卷中写入数据，然后再将这个存储卷挂载到主容器中。一个Pod可以拥有任意数量的init容器且init容器是顺序执行的，当且仅当最后一个init容器执行完毕之后才会启动主容器。如下所示

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: init-container
spec:
  containers:
    - name: app
      image: http-whoami
      volumeMounts:
        - name: www-data
          mountPath: /html
  initContainers:
    - name: first
      image: alpine
      command:
        - sh
        - -c
        - "touch /html/first && sleep 30"
      volumeMounts:
        - name: www-data
          mountPath: /html
    - name: second
      image: alpine
      command:
        - sh
        - -c
        - "touch /html/second && sleep 60"
      volumeMounts:
        - name: www-data
          mountPath: /html
  volumes:
    - name: www-data
      emptyDir:
        medium: Memory
```

通过init容器来延迟Pod主容器的启动，直到Pod的前置条件被满足为止。但是更好的做法是我们的应用不需要所依赖的组件都准备好才能启动（延时加载等），我们也可以通过**就绪探针来通知Kubernetes**当前应用还没准备好（依赖还没启动，循环等待），这样**能避免未就绪的应用成为服务的端点（Endpoint）且能在Deployment升级过程中生效避免错误版本出现**。

#### 增加生命周期钩子

除了通过init容器来介入Pod的启动过程之外，Pod还允许定义两种类型的生命周期钩子

* 启动后（`Post-Start`）钩子
* 停止前（`Pre-Stop`）钩子

这些生命周期钩子是基于单独的容器来指定的（init容器影响的是整个Pod），这些钩子与探针类似可以执行以下这些操作

* 在容器内部执行一个命令
* 向一个URL发送HTTP GET请求
* 发起一个Socket TCP连接

##### 使用生命周期钩子

**启动后钩子是在容器的主进程启动之后执行**的，它可以在应用启动时做一些额外的工作（比如执行一些通知或者初始化一些数据来让应用程序更顺利的运行）。**启动前钩子是与主进程并行执行的，如果钩子未执行完成退出则Pod处于`ContainerCreating`状态，且如果钩子执行失败或者返回了一个非零值，主容器会被直接杀死。**如果启动前钩子执行失败是无法在日志中看到详细信息，只能在Pod描述中看到一个`FailedPostSTartHook`事件（可以将钩子的输出内容写到容器的文件系统（持久卷）中）。

**停止前钩子是在容器被终止之前立即执行**，当一个容器需要终止运行时，kubelet会在配置了停止前钩子的容器上执行这个钩子，并且仅在执行完钩子程序后才会向容器进程发送`SIGTERM`信号（优雅终止）。

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: container-hook
spec:
  containers:
    - name: app
      image: laboys/http-whoami
      lifecycle:
        postStart:
          exec:
            command: # 未执行完成Pod处于`ContainerCreating`状态
              - sh
              - -c
              - "touch /pre-start && sleep 60"
        preStop:
          httpGet: # host的默认值是Pod的IP地址
            port: 8080
            path: /shutdown
```

#### 了解Pod的关闭

Pod的关闭是通过API服务器删除Pod对象里触发的，当API服务器接收都DELETE请求时，会给Pod设置一个`deletionTimestmp`值并触发事件更新。此时当kubelet发现订阅到要终止删除一个Pod时，它开始终止Pod中的每个容器（kubelet会给每个容器一定的时间来优雅地停止）。在**终止进程开始后定时器就开始运行**，接着按照顺序执行以下流程

* 执行停止前钩子（如果有的话），然后等待钩子执行完毕
* 向容器主进程发送`SIGTERM`信号
* 等待容器优雅地关闭或者直到时间超时
* 如果容器超时后还未关闭，则发送`SIGKILL`来强制杀死容器进程

kubelet在执行以上任意流程时超过了设定的等待时间，则会直接触发`SIGKILL`来杀死进程。

##### 设置关闭等待时间

关闭等待时间可以通过在YAML文件中指定`pod.spec.terminationGracePeriodSeconds`来设置（默认值为30），也可以在执行删除命令时通过添加`grace-period`参数来覆盖设置，也可以使用`--grace-period=0 --force`来强制删除一个Pod。

```bash
kubectl delete pod <pod-name> --grace-period=5
```

当Pod中所有的容器都停止后，Kubelet会通知API服务删除在etcd中的资源。

##### 在应用中合理地处理容器关闭操作

应用应该响应`SIGTERM`信号并在收到信号后执行关闭流程，除了可以监听`SIGTERM`信号之外，还可以通过停止前钩子接收关闭通知并执行关闭流程。

但无法确定一个Pod使能能在指定时间内完成关闭动作（比如关闭时要迁移大量的数据），我们可以在接收到关闭信号之后创建一个`Job`资源额外来处理关闭清理操作。更合理的方是用一个专门的持续运行的Pod（或者使用CrobJob资源）来检查是否需要进行清理任务。



### 确保所有的客户端请求都被正确关闭

我们不希望Pod在启动或者关闭过程中出现断开（无法）连接等情况导致服务中断，因为Kubernetes中并没有提供相对应的机制，所以需要在应用程序设计上遵循一些规则来避免遇到服务中断问题。

#### 在Pod启动时避免服务中断

为了在Pod启动时避免出现服务中断问题，我们只需要做到**给Pod添加就绪探针并当且仅当应用程序准备好处理请求时才让就绪探针返回成功**。

#### 在Pod关闭时避免服务中断

当API服务器接收到删除Pod的请求后，会标记这个Pod已被删除并发布事件通知给所有对此感兴趣的监听器（比如kubelet和端点控制器（Endpoint Controller））。当**Endpoint Coltroller**接收到这个事件后，他会从该Pod所在的所有服务中删除对应的Endpoint（向API服务器发起删除Endpoint请求），接下来**kube-proxy**会监听到有Endpoint资源被删除，然后在自己的节点上更新iptables规则以防止**新连接（不会影响已存在的连接）**被转发到这个Pod上。

在这个情况下，应用程序所需要做的是在接收到终止信号之后任然保持接收连接直到kube-proxy完成了对iptables的更新（实际上我们没办法知道什么时候处理完成了）。**应用程序唯一能做的是：在接收到终止信号之后等待几秒钟（等待kube-proxy删除转发规则），然后停止接收新连接（关闭Listen），开始逐步关闭已完成或不活跃的连接，并在所有活跃的请求处理完成后关闭应用**。

**我们至少可以添加一个停止前钩子来等待几秒钟在退出，这样甚至不需要修改代码**

```yaml
spec:
  containers:
    - name: web
      lifecycle:
        preStop:
          exec:
            command:
              - sh
              - -c
              - "sleep 5"
```



### 让应用程序在Kubernets中方便运行和管理

接下来我们需要构建方便在Kubernetes中管理的应用

#### 构建可管理的容器镜像

构建的应用镜像应该包括应用的可执行文件和它的依赖库，在这个基础上应用镜像应该尽可能的小而且不包含任何无用的东西（可以使用多阶段构建来缩减镜像的体积和保证镜像的纯净）。

#### 合理地给镜像打标签并正确使用ImagePullPolicy

在构建应用镜像时应该使用版本化的标签（甚至可以加上构建时间信息，只保留年月日即可），如果只使用`latest`标签会导致我们无法回退到之前的版本（除非我们重新推送了旧版本的镜像）。使用版本化的标签不仅有助于在Deployment，StatefulSet中快速回退版本，也有助于我们更好的观察Pod是否被更新。

当我们使用固定的标签（如`latest`）时，同时需要配置`pod.spec.imagePullPolicy`为`Always`，这会导致每次部署Pod都会尝试重新拉取镜像，会拖慢Pod的启动速度，同时这个策略会导致无法连接到镜像仓库时Pod无法启动。

#### 使用多维度的标签进行管理

我们最好给所有的资源（不仅仅是Pod）都打上标签，标签可以包含以下这些内容

* 资源所属的应用的名称
* 应用层级（后端、前端等等）
* 运行环境（开发、测试、预发布、生成等等）
* 版本号
* 发布类型（稳定版、金丝雀等）

标签管理可以让你以组的方式来管理资源，从而很容易了解资源的归属。

#### 通过注解描述资源

资源应该至少包含一个“描述资源”的注解和一个“描述负责人”的注解。在微服务系统中，每个微服务最好还可以包含一个注解用来“描述该服务所依赖的其他服务的名称”

#### 给进程终止提供更多信息

当容器持续终止运行的时候，我们可以把有关异常关闭的调试信息都写到日志中。在Kubernetes中有一个特性可以从Pod状态中很容易的看到容器终止的原因。我们可以向一个文件中写入详细的信息，随后这个消息就会被kubelet读取并显示在`kubectl describe pod <pod-name>`中。这个文件的默认路径是`/dev/termination-log`（也可以通过`spec.containers.terminationMessagePath`）进行修改

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: termination-message
spec:
  containers:
    - name: app
      image: alpine
      command:
        - sh
        - -c
        - "echo 'backtrace message from application' > /var/termination-reason; exit 1"
      terminationMessagePath: /var/termination-reason
      terminationMessagePolicy: FallbackToLogsOnError
```

以上`terminationMessagePath`表示需要读取的退出原因的文件名。而`terminationMessagePolicy`可以配置为`File`或者`FallbackToLogsOnError`。默认吗的`File`会从指定的文件中读取，而`FallbackToLogsOnError`则会在文件内容为空的情况下读取容器的最后几行日志当做终止消息（**只有在异常退出的情况下才会显示**）。

#### 处理应用日志

应用程序应该将日志写到标准输出力而不是文件中，这样就可以很容易的通过`kubectl logs`来查看应用日志。在生产环境中应该使用一个集中式的面向集群的日志解决方案，我们会将所有的容器日志收集并永久地存储到一个中心化得位置。常见的方案有`ELK(ElasticSearch, Logstash, Kibana)`等。

##### 处理多行日志

进行日志收集的时候一般都是一行一行进行读取并保存到数据库中的，当输出多行数据时就会导致一个日志被当做多个条目保存到数据库中，解决的方法也很简单，我们可以输出JSON格式的日志但是不利于用户查看。更佳的方案是同时输出到文件（JSON格式）和标准输出（可读性）上。



### 开发和测试的最佳实践

每个人都需要找到适合自己的最佳方式

#### 开发过程中在Kubernetes之外运行应用

开发一个应用时，我们不需要每次测试都在Kubernetes中运行，可以在自己的机器上进行开发和运行，如果应用程序依赖于Kubernetes中的一些功能，可以通过比如`NodePort`或者`kubectl proxy`来让我们在Kubernets之外使用集群内的服务。

#### 在开发过程中使用minikube

我们还可以在本地运行一个minikube集群并将应用程序放到本地集群中运行和测试。我们基于minikube还可以将本地文件通过minikube VM挂载到容器中进行热更新和测试。

#### 发布版本和自动部署资源

一个比较好的实践是将资源的manifest文件存放到一个版本控制系统中，这样可以方便做代码评审，审计追踪，或者是在任何需要的时候回退更改。通过还可以基于版本管理的Webhook甚至CICD来自动化将更改和新资源更新到集群中。
