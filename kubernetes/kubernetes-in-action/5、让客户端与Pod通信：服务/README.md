服务：让客户端发现Pod并与之通信
-------------------------------------------------

现在大多数的应用都需要根据外部请求作出响应，就微服务而言，Pod通常需要对来自集群内的其他Pod，或者来自外部的客户端程序的HTTP请求作出响应。在这种情况下Pod需要一种”寻找其他Pod“的方法，在Kubernetes中的每个应用配置文件中指出所依赖服务的精确地址或主机名这一方法是行不通的：

* **Pod不是持久化的**，它会随时的启动或者关闭，所以我们无法保证一个IP所指向的Pod一直存在
* **Kubernetes会在Pod启动前才分配IP地址**，所以无法提前知道提供服务的Pod的IP地址
* **水平伸缩意味着会有多个提供相同服务的Pod**，客户端无法知道Pod的数量以及对应的地址



### 介绍服务

Kubernetes服务是一种为”功能相同的Pod“**提供单一不变的接入点**的资源。当服务存在时，它的IP地址和端口不会改变。当客户端连接到服务时，服务会将客户端连接随机路由到提供该服务的一个Pod上。

#### 创建服务

我们可以选择使用命令行的方式创建服务

```bash
k expose rs http-whoami-rs --name http-whoami-srv --port=80 --target-port=8080
```

也可以通过创建yaml的方式来创建服务

```yaml
kind: Service
apiVersion: v1
metadata:
  name: http-whoami-srv
spec:
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: https
    port: 443
    targetPort: 8443
  selector:
    app: http-whoami
```

一个服务可以对应多个后端Pod，服务对所有进入的连接都是直接负载均衡到后端Pod上的。所以服务根据标签选择器来指定哪些Pod是对应的后端Pod。

##### 从内部集群测试服务

我们可以使用以下几种方式来检测一个服务：

* 创建一个Pod，这个Pod会访问对应的服务并将响应内容打印在标准输出上。我们通过观察标准输出来判断是否正常
* 使用ssh登录到其中一个Kubernetes节点上，然后使用curl进行测试
* 使用`kubectl exec`命令进入一个Pod的容器中，然后使用curl进行测试

对于第三种情况，我们可以使用`kubectl exec`在一个已存在的Pod中执行任何命令

```bash
k exec http-whoami-tester -- curl -vvv http://http-whoami-srv/
```

*这里使用双横线（--）表示kubenetes命令已经结束，之后的内容为需要执行的命令行和参数。如果不使用`--`进行隔断，可能会导致后续的参数被解析成exec的参数而产生歧义*

##### 配置服务商的会话亲和性

如果我们希望特定客户端产生的所有请求都指向同一个Pod，我们可以在在服务的`spec.sessionAffinity`属性上配置`ClientIP`（默认为`None`）。Kubernetes支持的会话亲和性仅有`ClientIP`和`None`，这是因为服务是IP层的负载均衡，需要同时处理TCP和UDP请求且无法解析连接中的内容（所以没有基于`Cookie`的亲和性）。

##### 同一服务暴露多个端口

服务可以一次性暴露多个端口，**但是必须指定每个端口的名字（通过`spec.ports.name`字段）**。

**注意：服务的标签选择器是应用于整个服务的，不能根据不同的端口使用不同的选择器（只能使用多个服务）。**

##### 使用命名端口

我们可以在Pod上通过命名端口的形式解耦与端口号与服务之间的绑定关系，并且可以提供给给服务更好的可读性

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: http-whoami
  labels:
    app: http-whoami
spec:
  containers:
  - name: app
    ports:
    - name: http
      containerPort: 8080
    - name: https
      containerPort: 8443
---
kind: Service
apiVersion: v1
metadata:
  name: http-whoami-srv
spec:
  selector:
    app: http-whoami
  ports:
  - name: http
    port: 80
    targetPort: http
  - name: https
    port: 443
    targetPort: https
```

在以上yaml中，我们在Pod中命名了2个端口这样最大的好处是我们可以**随意修改Pod中的端口号而无需修改服务的相关属性**。



### 服务发现

Kubernetes为客户端提供了发现服务的IP地址和端口的方式。

#### 通过环境变量发现服务

在Pod开始运行的时候，Kubernetes会初始化一系列的环境变量指向现在存在的服务（需要服务在Pod创建前就已经存在），如下

```bash
KUBERNETES_PORT=tcp://10.96.0.1:443
KUBERNETES_SERVICE_HOST=10.96.0.1
KUBERNETES_SERVICE_PORT=443
KUBERNETES_SERVICE_PORT_HTTPS=443
KUBERNETES_PORT_443_TCP=tcp://10.96.0.1:443
KUBERNETES_PORT_443_TCP_ADDR=10.96.0.1
KUBERNETES_PORT_443_TCP_PORT=443
KUBERNETES_PORT_443_TCP_PROTO=tcp

HTTP_WHOAMI_PORT=tcp://10.96.196.76:80
HTTP_WHOAMI_SERVICE_HOST=10.96.196.76
HTTP_WHOAMI_SERVICE_PORT=80
HTTP_WHOAMI_SERVICE_PORT_HTTP=80
HTTP_WHOAMI_PORT_80_TCP=tcp://10.96.196.76:80
HTTP_WHOAMI_PORT_80_TCP_ADDR=10.96.196.76
HTTP_WHOAMI_PORT_80_TCP_PORT=80
HTTP_WHOAMI_PORT_80_TCP_PROTO=tcp
```

**需要注意的是服务名称中的横杠被转换为下划线，并且当服务名称作为环境变量名称的前缀时，所有的字母都是大写的**

#### 通过DNS发现服务

在kube-system命名空间下有一个名为kube-dns的Pod，这个Pod运行DNS服务，在集群中的其他Pod的都会将它作为DNS（通过`/etc/resolve.conf`实现）。运行在Pod中的进程在进行DNS查询时都会被Kubernetes自身的DNS服务器响应，该服务知道系统中运行的所有服务。

#### 通过FQDN连接服务

在Pod中我们可以通过`FQDN`（全限定域名）访问某一个指定的服务（在这种情况下任需要提供端口，但是可以从环境变量中获取）。

```
SVCNAME.NAMESPACE.svc.cluster.local
```

如果客户端Pod与服务端Pod处于同一命名空间，我们可以直接省略`.svc.cluser.local`后缀，甚至可以省略命名空间段。我们可以在一个已经启动的Pod容器中通过`curl`命令使用FQDN来访问服务。

```bash
k exec -ti busybox sh
```

**需要注意的是，我们无法ping到这个服务指向的IP，这是因为服务的集群IP是一个虚拟IP，这个IP只有在于服务端口结合时才有意义。**
