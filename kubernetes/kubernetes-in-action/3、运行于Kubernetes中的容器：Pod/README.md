运行于Kubernetes中的容器：Pod
-----------------------------------------------

**Pod是Kubernetes中最为重要的核心概念**，而其他对象仅仅在管理、暴露Pod或被Pod使用。



### 介绍Pod

Pod是一组容器的组合，代表了Kubernetes中的基本构建模块。当一个Pod包含多个容器时，这些**容器总是运行于同一个工作节点**上（一个Pod绝不会跨越多个工作节点）。

#### 为何多个容器比单个容器多个进程好

**容器被设计为每个容器只运行一个进程**（除非进程本身产生子进程）。如果在单个容器中运行多个进程，那么保持所有进程运行，管理它们的日志等都将会是我们的责任。



#### 了解Pod中共享和隔离的资源

由于不能将多个进程聚集在一个单独的容器中，我们就需要另一种更高级的的机构来将容器绑定在一起，并将它们作为一个单元进行管理，这就是Pod出现的根本原因。

在Pod下，我们可以运行一些密切相关的进程，并为它们提供（几乎）相同的环境，而且这些进程**好像全部运行于单个容器中，同时又保持一定的隔离**。这样一来我们就可以全面利用容器所提供的特性，同时对这些进程来说就像运行在一起一样，实现两全其美。

##### Pod中容器之间的资源共享

因为Kubernetes中管理的基本单位是Pod，所以我们需要**每个容器组之间共享一些资源**。Kubernetes通过配置Docker来让**一个Pod内的所有容器共享相同的Linux命名空间**。

* Pod下的**容器都在相同的Network和UTS命名空间**下（共享相同的主机名和网络接口）
  * 需要注意同一Pod下的容器运行的进程不能绑定到相同的端口号（端口冲突）
  * 同一Pod下的容器具有相同的loopback网络接口，因此容器可以通过localhost与同一Pod的其他容器进行通信。
* Pod下的**容器都在相同的IPC命名空间**下运行，因此他们能通过IPC进行通信
* *在最新的Kubernetes中，它们特能共享相同的PID命名空间（默认未激活）*
* 默认情况下，**每个容器的文件系统与其他容器完全隔离**（但是可以使用名为Volume的Kubernetes资源来共享文件目录）

##### Pod之间的平坦网络

Kubernetes集群的所有Pod都在同一个共享网络地址空间中，这意味着每个Pod都可以通过其他Pod的IP地址来实现相互访问。不论是将两个Pod安排在单一的还是不同的工作节点上，同时不管实际节点的网络拓扑结构如何，这些Pod内的容器都能够像在无NAT的平坦网络中一样相互通信。

#### 通过Pod合理管理容器

因为Pod比较轻量，我们可以在几乎不产生任何额外开销的前提下拥有尽可能多的pod。与将所有内容填充到一个Pod中不同，我们应该将应用程序组织到多个Pod中，而每个pod只包含紧密相关的组件或进程。

##### 将多层应用分散到多个Pod中

对于有多层的应用程序（如web应用有前端服务、后端服务、基础服务）的每个层分散到不同的Pod中，这样有利于Kubernetes将Pod拆分到不同的工作节点上。从而提高基础架构的利用率。

##### 基于扩缩容考虑而分割到多个Pod中

另一个不将应用程序都放到单一Pod中的原因就是扩缩容。Pod也是扩缩容的基本单位，对于Kubernetes来说，它不能横向扩容单个容器，只能扩缩容整个Pod。

通常来说每种类型的Pod应该都有不同的缩放策略和需求，所以我们倾向于分别独立地扩缩它们。

##### 何时在Pod中使用多个容器

需要在一个Pod中部署多个容器的主要理由是应用程序是由一个主进程和多个辅助进程组成。例如Sidecar模式就需要在一个Pod中部署其他容器来帮助我们进行：日志收集、数据处理、通信适配等。



### 以YAML或JSON描述文件创建Pod

Pod和其他Kubernetes资源通常是通过向Kubernetes REST API提供JSON或YAMl描述文件来创建的。通过YAML文件定义所有的Kubernetes对象之后，还可以将他们存储在版本控制系统中，充分利用版本控制所带来的便利性。

#### 查看现有Pod的YAML描述文件

我们可以使用如下命令查看现有Pod的YAML描述文件

```bash
kubectl get pods xxx-yyy -o yaml

apiVersion: v1
kind: Pod
metadata:
  ...: ...
spec:
  ...: ...
status:
  ...: ...
```

Pod定义由这么几部分组成：首先是YAML中使用的Kubernetes API版本和YAML描述的资源；其次是几乎所有Kubernetes资源中都可以找到的三大重要部分：

* `matedata`：包括名称、命名空间、标签和关于该容器的其他信息
* `spec`：包含Pod内容的实际说明，例如Pod的容器、卷和其他数据
* `status`：包含运行中的Pod的状态信息，例如Pod所处的条件、每个容器的描述和状态、以及内部IP和其他基本信息
  * **在创建Pod时我们永远不需要提供这个部分**

#### 通过YAML创建一个Pod

首先需要编写对应的Yaml文件：

```yaml
apiVersion: v1 // 遵循v1版本的Kubernetes API
kind: Pod // 描述的是一个Pod资源
metadata: // Pod的元数据
  name: node-hostname // Pod的名字
spec: // Pod的规格
  containers: // Pod中的容器列表
  - image: node-hostname:latest // 当前容器所使用的镜像
    name: node-app // 该容器的名词
    ports: // 该容器对外暴露的端口（**仅仅只是展示性的**）
    - containerPort: 8080
      protocol: TCP
```

对于以上字段我们可以使用如下命令来查询手册对字段的解释和样例

```bash
kubectl explain pod
kubectl explain pod.spec
```

有了以上yaml文件之后，我们可以通过以下命令创建Pod

```bash
kubectl create -f node-hostname.yaml
```

#### 查看Pod日志

我们可以通过如下命令查看Pod的日志（类似Docker中查看容器日志的方式）：

```bash
kubectl logs node-hostname
kubectl logs node-hostname -f # 以流的模式查看

kubectl logs node-hostname -c node-app # 使用-c参数指定Pod中的某个容器
```

**需要注意的是：每天或者每次日志达到10MB大小之后，容器日志都会自动轮替（rotate）。`kubectl logs`命令仅显示最后一次轮替后的日志内容。**

#### 向Pod发送请求

除了service的方式之外，还有其他连接到Pod并进行测试和调试的方法，其中之一便是**端口转发**。使用如下命令创建端口转发

```bash
kubectl port-forward node-hostname 8088:8080 # 将数据从本地的8088端口转发到node-hostname的8080端口，与Docker顺序一致
```



### 使用标签组织Pod

随着Pod数量的增多，如果没有可以有效组织这些组件的机制，将会导致产生巨大的混乱。很明显，我们需要一种能够基于任何标准将Pod组织成更小子集的方式，这样一来处理系统的每个开发人员和系统管理员都可以轻松地看到哪个Pod是什么。

#### 标签

标签是一种简单却功能枪法的Kubernetes特性，不仅可以组织Pod，也可以组织所有其他的Kubernetes资源。详细的说，标签就是一个可以给附加到资源上的键值对，用于在选择资源时通过标签确定一组资源。

一个资源可以拥有多个标签，通常在创建资源时就会将标签附加到资源上，也可以在创建资源后添加或者修改标签的内容。

#### 创建资源时指定标签

在创建资源时，可以在YAML文件中`metadata.labels`中指定标签列表

```yaml
kind: Pod
apiServer: v1
metadata:
  name: http-hostname
  labels:
    app: http-hostname
    env: production
spec:
  containers:
  - name: app
    image: http-hostname
    ports:
      - containerPort: 8080
        protocol: TCP
```

使用`kubectl apply -f x.yaml`后，我们可以使用以下命令查看标签

```bash
kubectl get pods --show-lables # 查看Pod的所有标签
kubectl get pods -L app,env # 查看指定的标签
```

#### 修改现有资源的标签

标签可以在现有资源上添加或者修改而无需重新创建资源

```bash
kubectl label pod http-hostname company=foo,department=it # 新增标签
kubectl label pod http-hostname env=development --overwrite # 修改标签需要额外的--overwrite参数
```



### 通过标签选择器列出Pod子集

标签选择器允许我们选择标记有特定标签的pod子集，并对这些pod执行操作。标签选择器可以根据资源的以下条件来选择资源：

* 包含（或者不包含）使用特定键的标签：`-l 'key-includede' -l '!key-excluded'`
* 包含具有特定键和值的标签：`-l key=value`
* 包含具有特定键的标签，但其值与我们指定的不同：`-l key!=value`
* 包含具有特定键的标签，其值与我们指定的任一一个相同：`-l 'key in (value1,value2)'`
* 包含具有特定键的标签，其值与我们指定的都不同：`-l 'key notin (value1,value2)'`

可以使用逗号分隔多个条件表示需要同时满足（且的关系），使用多个`-l`参数来分隔多个条件表示只需要满足一个即可（或的关系）

```
k get po -l 'app in (db,mq),env=prod' -l 'app notin (db,mq),env!=prod'
// ((app in [db, mq]) AND env = prod) OR ((app notin [db, mq]) AND env != prod)
```



### 使用标签和选择器来约束Pod调度

在Kubernetes中，我们创建的Pod都是近乎随机地调度到工作节点上，而在某些情况下，我们希望将Pod调度某些特定的节点上。在这种情况下下我们不应该直接指定一个确切的节点，而是应该使用标签和节点选择器来描述对节点的需求。

```bash
k label node apple gpu=true
k label node banana ssd=true
```

通常来说，当运维团队在向集群中添加新节点时，会通过附加标签来对节点进行分类，这些标签指定节点提供的硬件类型或者节点上对调度pod能提供便利的其他信息。

#### 使用标签选择器

接下来我们就可以在yaml文件中通过声明`nodeSelector`来选择我们所需要调度到的特定节点

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: http-whoami
  labels:
    app: http-whoami
spec:
  nodeSelector:
    ssd: "true"
    gpu: "false"
  containers:
  - name: app
    image: http-whoami
```

#### 调度到特定的节点

我们也可以通过每个节点都有的唯一标签`kubenetes.io/hostname`（该节点的实际主机名）来选择某个具体的节点。**这是不推荐的，除非你知道你在做什么**



### 注解Pod

除标签之外，Pod以及其他对象还可以包含注解。注解也是键值对，与标签本质上非常相似。不过注解并不是为了对资源分组而是为了保存标识等帮助信息而存在的。Kubernetes也会将一些注解自动添加到对象，但是其他注解需要用户手动添加。大量使用注解可以为每个资源添加说明，以便每个使用该集群的人都可以快速查找有关该资源的信息。

#### 查找对象的注解

为了查看对象的注解我们需要查看对象完整的yaml或者使用describe来查看

```bash
k get po poname -o yaml
k describe po poname
```

#### 添加和修改注解

和标签一样民主街可以在对象创建时就添加，也可以通过`k annotation`命令（与`k label`语法基本一致）添加到现有对象中

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: poname
  labels:
    app: poname
  annotations:
    laboys.io/creator: jayson
...: ...
```



### 使用命名空间对资源进行分组

当需要将对象分割到完全独立且不重叠的组中，每次只想在一个组内进行操作，可以使用Kubernetes的对象命名空间。Kubernetes命名空间简单地为对象名称提供一个作用域，此时我们可以将对象组织到多个命名空间中（在不同命名空间中可以使用相同的资源名称）。

*命名空间为**资源名称**提供一个作用域，除了隔离资源，命名空间还可以用于限制某些用户可以访问的资源，甚至限制单个用户可用的计算资源数量。*

#### 发现其他命名空间及其对象

当使用`k get`命令列出资源时，如果未指定命名空间则默认使用的是`default`命名空间，只会显示这个命名空间下的对对象。命名空间使我们能够将一组不相关的资源分到不重叠的组中，有助于每个操作人员各自管理自己的资源集合。

#### 创建一个命名空间

创建命名空间可以通过将yaml文件提交给Kubernetes API服务来实现

```yaml
kind: Namespace
apiVersion: v1
metadata:
  name: custom-namespace
```

也可以通过`k create namespace custom-namespace`命令来创建

#### 管理命名空间中的对象

如果想在其他命名空间中创建对象，我们可以在对象的`metadta.namespace`字段上指定命名空间，或者在命令中使用`-n`参数指定。

#### 命名空间提供的隔离

尽管命名空间可以将对象分隔到不同的组，只允许用户对属于特定命名空间的对象进行操作，但实际上命名空间之间不提供运行中的资源的任何隔离。不同命名空间之间的资源都可以相互访问。



### 停止和移除资源

当我们不需要Kubernetes中运行的一些对象时，我们可以按照名称来删除资源

```bash
k delete pod podname
```

上述命令会指示Kubernetes终止该Pod中的所有容器，Kubernetes会向这些容器发送一个`SIGTERM`信号并等待一段时间（默认为30s）让应用正常关闭。如果没有正常关闭，Kubernetes会发送`SIGKILL`信号来强制终止该进程。

或者我们可以通过标签选择器来删除资源

```bash
k delete pod -l rel=canary
```

也可以通过删除命名空间的方式来删除资源（对象会随着命名空间删除而删除）

```bash
k delete ns custom-namespace
```

删除命名空间中的所有Pod，但保留命名空间可以使用如下命令（`-all`参数告诉Kubernetes删除当前命名空间中的所有Pod）

```bash
k delete pod --all
```

甚至我们可以直接删除当前命名空间下的所有对象

```bash
# all表示删除所有的资源类型
# --all表示删除所有的资源实例
k delete all --all
```

