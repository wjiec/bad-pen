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

