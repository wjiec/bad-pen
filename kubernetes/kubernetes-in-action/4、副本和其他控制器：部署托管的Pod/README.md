副本机制和其他控制器：部署托管的Pod
-------------------------------------------------------

Pod是Kubernetes中的基本部署单元，在实际的用例中，我们**希望部署的Pod能自动保持运行并且保持健康而无需任何手动的干预**。如果我们直接创建Pod，但是节点在之后的某个时间崩溃了，那么节点上的Pod会丢失，并且不会被新节点替换，除非这些Pod是由一些控制器管理的。

### 保持Pod健康

使用Kubernetes的一个主要好吃是，可以**给Kubernetes一个容器列表并由其来保持容器在集群中存活**。只要Pod调度到某个节点，该节点上kubelet就会运行Pod的容器，从此只要该Pod存在，kubelet就会保持他们一直存活。**kubelet可以自动重启主进程崩溃的容器**，但是如果**应用程序因为死循环或者死锁而停止响应**，为了确保kubelet在这种情况下可以检测到并重新启动容器，我们**必须从外部检查容器的运行情况，而不是依赖于应用的内部检测**。

#### 存活探针

Kubenertes可以通过存活探针（Liveness probe）检查容器是否还在运行。Kubernetes有以下三种探测容器的机制：

* `HttpGet`：对容器的IP地址执行HTTP GET请求，如果收到响应且状态码为2xx或者3xx则探测成功，否则认为是失败的。
* `TCP`：对容器内指定端口建立TCP连接，如果连接成功则探测成功。
* `Exec`：在容器内执行任意命令，如果命令的退出状态码为0则探测成功。

#### 创建基于HTTP的存活探针

探针可以在yaml中的`spec.containers.livenessProbe`段来指定，如下

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: http-whoami
spec:
  containers:
  - name: app
    image: http-whoami
    env:
    - name: SERVICE_HEALTHY_COUNT
      value: "5"
    livenessProbe:
      httpGet:
        path: /
        port: 8080
```

该描述定义了一个`HttpGet`的探针，kubelet会在容器的8080端口上执行`GET /`请求以确定容器是否健康。

#### 探针的附加属性

还有一些额外的属性可以用来控制探针的行为：

* `initialDelaySeconds`：在容器启动多久之后开始执行第一次探针
* `timeoutSeconds`：每次探针执行的超时时间（等待响应时间）
* `periodSeconds`：每次探针的间隔时间
* `failureThreshold`：允许连续失败几次才进行重启操作

```yaml
spec:
  containers:
  - name: app
    livenessProbe:
      initialDelaySeconds: 30
      failureThreshold: 10
```

如果没有设置初始延迟，探针将在启动时立即开始探测容器，这通常会导致探测失败。所以**务必记得设置一个初始延迟来说明应用程序的启动时间**。

#### 创建有效的存活探针

对于在生产中运行的Pod，一定要定义一个存活探针。没有探针的话Kubernetes无法知道应用是否还活着。

简易的探针仅仅检查了服务器是否还在响应请求，这在大多啥情况下可能已经足够了。

* **只检查应用程序的内部**：不要加入任何其他外部因素的干扰（如web程序不应该因为数据库连接失败而报告失败）
* **保持探针的轻量**：探针不要消耗太多的计算资源，因为这些资源也是算在容器的资源配额里的
* **无需在探针中实现重试**：探针的失败预制是可设置的，在探针中实现重试循环是浪费资源



### ReplicationController

ReplicationController是一种Kubernetes资源，它可以确保托管于它的Pod始终保持运行状态。如果Pod因任何原因消失（例如节点从集群中消失或者Pod被人意外删除），则ReplicationController会创建一个新的替代Pod。

一般而言，ReplicationController作用是创建和管理一个Pod的多个副本（Replicas），这就是ReplicationController名字的由来。

#### ReplicationController的操作

Rc会持续监控正在运行的Pod列表，并保证对应“选择器”的Pod数量与期望相符。Rc的工作是确保Pod的数量始终与其标签选择器相匹配。Rc主要由三部分组成：

* `Label Selector`：标签选择器用于确定Rc的工作范围
* `Replica Count`：副本数用于指定应该要运行的Pod数量
* `Pod Template`：Pod模板用于创建新的Pod副本

Rc的副本数量、标签选择器甚至Pod模板都是可以随意修改的，但是**只有副本数量的变更会影响现有的Pod**。

* 修改标签选择器只会让某些Pod“脱离”Rc的视线范围，控制器将会停止关注原有的Pod，所以不会影响现有的Pod。
* 修改Pod模板只会对下一次新建的Pod产生影响而不会导致现有Pod发生变化。

##### 使用ReplicationController的好处

ReplicationController虽然是一个非常简单的概念，但却提供了以下强大的功能：

* 确保一个Pod（或者多个副本）持续运行，在Pod丢失后重新创建一个
* 集群节点发生故障时，它将俄日故障节点上受Rc管理的Pod创建副本
* 能轻松实现Pod的水平伸缩（手动或者自动）

#### 创建一个ReplicationController

与其他资源一样，我们可以通过在一个yaml文件声明Rc的方式来创建

```yaml
kind: ReplicationController
apiVersion: v1
metadata: http-whoami-rc
spec:
  selector:
  	app: http-whoami
  replicas: 3
  template:
  	metadata:
  	  labels:
  	    app: http-whoami
  	spec:
  	  containers:
  	  - name: app
  	    image: http-whoami
  	    ports:
  	    - containerPort: 8080
  	    livenessProbe:
  	      httpGet:
  	        path: /
  	        port: 8080
  	      initialDelaySeconds: 5
```

这里主要注意我们在`spec`中声明了`replicas`为当前期望的副本数量，`selector`则是表明Rc的监控范围，`template`则是对应Pod的模板。

**显然模板里的`template.metadata.labels`需要与`selector`相匹配，不然控制器将无休止的创建新Pod（新创建的Pod不在控制器的监控范围）**

**不指定`selector`会是一个更好的选择，这样Kubernetes将会根据`template`中的内容自动生成标签选择器**

**当`selector`中存在多个标签选择器时，需要同时满足所有的选择器，也就是“且”（AND）的关系**

#### 使用ReplicationController

在我们删除一个Rc创建的Pod时，Rc会对我们的行为作为响应，即创建一个新的Pod来满足“期望”。虽然Rc会收到我们删除Pod的消息通知，但这不是它创建新Pod的原因。这个**通知会触发控制器检查实际的Pod数量与期望值是否相同而是否需要采取措施**。

##### 应对节点故障

在节点发生故障之后，Kubernetes不会立即检测节点是否下线（这有可能只是瞬间的网络中断或者kubelet重启），而是在等待一段时间后才将节点标记为`NotReady`。这时候所有调度到该节点的Pod的状态将会修改为`Unknown`，这时Rc将会做出响应创建替代Pod。

##### 将Pod移出或移入ReplicationController的作用域

在任何时刻，Rc管理的是与标签选择器相匹配的Pod，可以通过修改Pod的标签将它从Rc的作用域中删除或添加。

**尽管Pod没有显式的绑定到Rc上，但是在被Rc管理的Pod的`metadata.ownerReferences`里会有一个字段引用着Rc，这有助于简单的找到这个Pod被哪个控制器所管理。需要注意的是，如果将一个Pod从Rc的管理中移除，响应的字段也会消失。**

使用案例：如果知道某个Pod发生了故障，就可以将它从Rc的管理范围中移除，这是控制器会创建一个新的Pod以代替他，而移除的Pod就可以随意调试检查，完成后删除Pod即可。

##### 修改Pod模板

Rc的Pod模板可以随时修改，这并不会影响当前已存在的Pod。只有在Rc需要创建新Pod时，修改的模板才会生效。

##### 水平伸缩Pod

放大和缩小Pod的数量规模其实就是修改ReplicationController中的`spec.replicas`的值。我们可以使用以下几种方式进行修改

```bash
# 直接编辑yaml文件形式
k edit rc http-whoami-rc

# 使用伸缩命令形式
k scale rc http-whoami-rc --replicas=10
```

##### 删除一个ReplicationController

当通过`k delete rc ...`命令删除一个Rc时，Pod也会被一起删除，我们也可以选择通过在命令中添加`--cascade==orphan`做到只删除Rc而不删除Pod。

我们可以使用适当的标签选择器创建新的Rc，并再次管理那些没有被一起删除的Pod。

### 

### 使用ReplicaSet而不是ReplicationController

Kubernetes后来又引入了一个名为ReplicaSet的资源，它是新一代的ReplicationController，并且将在未来完全替换掉Rc。通常情况下我们不会直接创建Rs，而是在创建更高层级的Deployment资源时自动创建它们。

#### 比较ReplicaSet和ReplicationController

**ReplicaSet的行为与ReplicationController完全相同**，唯一的不同是Rs的标签选择器的表达能力更强（比如Rc无法基于是否存在标签来选择Pod，而Rs可以）。

#### 定义ReplicaSet

与其他所有资源一样，我们通过yaml定义Rs

```yaml
kind: ReplicaSet
apiVersion: apps/v1
metadta:
  name: http-whoami-rs
spec:
  replicas: 3
  selector:
    matchLabels:
      app: http-whoami
    matchExpressions:
    - key: app
      operator: In
      values:
      - "http-whoami"
      - "whoami-http"
  template:
    metadata:
      labels:
        app: http-whoami
    spec:
      containers:
      - name: app
        image: http-whoami
```

以前内容与Rc最大的区别在于`spec.selector`部分，我们不直接在`selector`中直接指定标签，而是在另外的`matchLabels`和`matchExpressions`中指定。

#### 使用ReplicaSet更有表达力的标签选择器

Rs相对于Rc的主要改进是它更具表达力的标签选择器。我们可以通过`matchExpressions`添加额外的表达式，**每个表达式都必须包含`key`和一个`operator`（运算符），并且可能包含一个额外的`values`字段**。可用的`operator`有以下4个

* `In`：表示标签的值必须匹配`values`中的任一一个
* `NotIn`：表示标签的值与`values`中的值都不匹配
* `Exists`：表示存在某个标签（不需指定`values`）
* `DoesNotExists`：表示不得包含某个标签（不需指定`values`）

**需要注意的是如果指定了多个表达式，则所有表达式都需要满足才可以（Rc的`selector`同样如此）。**



### 使用DaemonSet在节点上运行Pod

Rc和Rs都用于在Kubernetes集群上运行部署特别数量的Pod。但是在需要在集群中的每个节点上运行一些Pod用于执行系统级或与基础架构相关的操作（例如日志收集器和资源监视器）时我们可以使用DaemonSet。

#### 使用DaemonSet在每个节点上运行一个Pod

DaemonSet没有期望的副本数的概念，如果节点下线，DaemonSet不会在其他地方重新创建Pod。而当一个新节点加入集群是，DaemonSet会立即部署一个新的Pod实例在这个节点上。

DaemonSet可以通过在Pod模板中指定`nodeSelector`的方式来选择只在部分节点上运行实例。**注意：虽然在集群中可能某些节点被设置为不可调度的，但是Daemon可以将Pod部署到这些节点上，因为不可调度这个属性只会被调度器所使用，而DaemonSet管理的Pod则完全绕过了调度器。**

#### 创建一个DaemonSet

与之前的所有资源一样，我们通过一个yaml文件来创建一个DaemonSet

```yaml
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: ssd-monitor
spec:
  template:
    metadata:
      labels:
        app: ssd-monitor
    spec:
      containers:
      - name: app
        image: nginx
      nodeSelector:
        disk: ssd
```

当我们向节点添加或者删除`disk: ssd`标签时，DaemonSet会进行相应的Pod创建和删除操作。

