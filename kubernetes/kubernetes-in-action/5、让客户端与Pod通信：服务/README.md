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



### 连接集群外部的服务

Kubernetes可以通过Service的服务特性暴露外部服务（服务将被重定向到集群外的IP，而不是重定向到内部IP），这样可以充分利用服务（指Kubernetes的Service资源）的负载均衡和服务发现能力。

#### 服务Endpoint

服务并不是直接与Pod相连的，而是由Endpoint资源介于两者之间。Endpoint资源就是暴露一个服务的IP地址和端口的列表，Endpoint资源和其他Kubernetes资源一样，可以通过`k get|describe`来获取它的基本信息。

**注意：在服务中定义的`spec.selector`选择器不是在重定向时使用的，而是用于构建IP和端口，然后存储于Endpoint资源中。当客户端发起连接请求时，服务代理程序从Endpoint的IP地址和端口列表中选择一个将其作为重定向的目标。**

#### 手动配置Endpoint

如果我们在创建服务时不指定Pod选择器，Kubernetes将不会创建Endpoint资源。Endpoint是一个单独的资源并不是服务的一个属性，而且Endpoint需要与服务具有相同的名字，并包含该服务的目标IP地址和端口列表

```yaml
kind: Service
apiVersion: v1
metadata:
  name: http-whoami
spec:
  ports:
  - port: 80
---
kind: Endpoint
apiVersion: v1
metadata:
  name: http-whoami
subsets:
  - addresses:
    - ip: 172.16.0.1
    - ip: 172.16.0.6
    ports:
    - port: 80
```

#### 为外部服务创建别名

除了手动配置服务的Endpoint来代替公开外部服务之外，我们还可以通过完全限定域名来访问外部服务。要使用这种类型的服务，我们需要指定服务的类型为`ExternalName`，并在`externalName`中使用完全限定域名来指定目标。

```yaml
kind: Service
apiVersion: v1
metadata:
  name: search-engine
spec:
  type: ExternalName
  externalName: bing.com
  ports:
  - port: 80
```

服务创建完成之后，Pod就可以通过`search-engine.default.svc.cluster.local`（或者`search-engine`）来访问外部服务。通过这个方法我们隐藏了实际的服务名称，并且允许服务在之后将其修改为指向其他位置的服务，或者是修改回`ClusterIP`类型并制定Pod选择器，甚至可以是手动创建的Endpoint。

**注意：在使用`ExternalName`类型时，DNS将通过新增一条CNAME记录的方式来重定向连接。这时候客户端将直接连接到外部服务，并将完全绕过服务代理。**



### 将服务暴露给外部客户端

当我们需要向外部暴露公开某些服务时，我们有以下几种方式可选：

* 将服务的类型设置为`NodePort`：Kubernetes会在集群的每个节点上打开一个端口（可指定或不指定），任一节点上该端口的访问的连接将会被路由到Pod中。
* 将服务的类型设置为`LoadBalance`：相当于在`NodePort`基础上再套一层负载均衡（由服务商提供，当不存在时表现和`NodePort`一致）
* 创建一个`Ingress`资源，这是构建于应用层之上的路由（根据协议进行路由具体的Pod）

#### 使用NodePort类型的服务

NodePort类型的服务可以让Kubernetes在集群的所有节点上打开一个端口（所有节点使用相同的端口号），并将所有节点上的连接转发给具体的Pod。

```yaml
kind: Service
apiVersion: v1
metadata:
  name: http-whoami-np
spec:
  type: NodePort
  ports:
  - port: 80
    targetPort: 8080
    nodePort: 30123
  selector:
    app: http-whoami
```

**注意：`spec.ports.nodePort`并不是强制的，如果没有这个字段，Kubernetes将随机选择一个端口**

在这种情况下如果我们只在配置文件中配置某一个节点的端口地址（`node1:30123`）这又回到了最初的样子。**如果该节点意外宕机后将无法再通过node1访问到该服务**

#### 通过负载均衡器将服务暴露出来

为了解决上述NodePort的问题，Kubernetes支持从云基础服务商那里自动获取一个负载均衡器（由这个负载均衡选择一个NodePort进行访问）。而我们需要做的仅仅是将服务的类型修改为`LoadBalancer`。

```yaml
kind: Service
apiVersion: v1
metadata:
  name: http-whoami-np
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
#    nodePort: 30123
  selector:
    app: http-whoami
```

**注意：如果Kubernetes在不支持LoadBalancer服务的环境中运行，则不会有创建负载均衡器的操作，这种情况下LoadBalancer与NodePort的表现是一致的。**所以这就是为什么说“LoadBalancer服务是NodePort服务的扩展”的原因。、

#### 外部连接的一些特性

在NodePort服务环境下，当客户端连接到达某一节点时，连接将被随机转发到其他（或本地）的Pod上。如果连接被转发到其他节点的Pod上将会增加额外的网络跳数，并且会**因为发生了SNAT导致丢失原始连接的客户端IP和端口信息**。

为了防止这种情况出现我们可以配置`spec.externalTrafficPolicy`的值为`Local`来解决这个问题

```yaml
kind: Service
apiVersion: v1
metadata:
  name: http-whoami-np
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: http-whoami
  externalTrafficPolicy: Local
```

**注意：当配置该属性值为Local时，如果当前节点上并没有对于的Pod运行那么客户端的连接将被挂起**



### 通过Ingress暴露服务

> Ingress（名词）—— 进入或进入的行为；进入的权利；进入的手段或地点；入口。

#### 为什么需要Ingress

一个重要的原因是每个LoadBalancer服务都需要自己的负载均衡器以及独有的公有IP地址，而Ingress只需要一个公网IP地址就能为很多服务提供访问（Ingress会根据请求的主机名和路径决定将请求转发给哪个服务）

Ingress在应用层（HTTP）之上执行操作（也可以工作在传输层上进行tcp/udp代理），并且可以提供一些服务不能实现的功能（实现基于cookie的会话亲和性）

#### 创建Ingress资源

**需要注意的是，如果需要使用Ingress资源，则Kubernetes中必须有Ingress控制器在运行**

```yaml
kind: Ingress
apiVersion: networking.k8s.io/v1
metadata:
  name: http-whoami
spec:
  rules:
  - host: http-whoami.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: http-whoami
            port:
              name: http
```

#### 通过Ingress访问服务

我们根据配置文件所定义的域名，通过curl发起请求进行测试

```bash
curl -vvv -H 'Host: http-whoami.example.com' http://localhost
```

#### Ingress的工作原理

客户端首先进行DNS查询获取到Ingress控制器的IP地址，然后像控制器发起请求，并在Host头中指定想要访问的域名，而控制器根据这个头部确定与之对应的服务，并读取服务中的Endpoints并从中选择一个Pod，然后将请求转发过去。

**注意：Ingress控制器并不会将请求转发给服务，只是用它来选择一个Pod。**

#### 在一个Ingress中暴露多个服务

由于Ingress定义中`rules`和`paths`都是数组，所以我们可以在一个Ingress中声明多个域名和多个路径暴露多个服务

```yaml
kind: Ingress
apiVersion: networking.k8s.io/v1
metadata:
  name: multi-service
spec:
  rules:
    - host: foo.example.com
      http:
        paths:
          - path: /order
            backend:
              service:
                name: order-service
                port:
                  name: http
          - path: /user
            backend:
              service:
                name: user-service
                port:
                  name: http
    - host: bar.example.com
      http:
        paths:
          - path: /product
            backend:
              service:
                name: product-service
                port:
                  name: http
          - path: /notification
            backend:
              service:
                name: notification-service
                port:
                  name: http
```

控制器可以根据URL中的路径将其转发到不同的服务里，也可以根据请求的域名来转发到不同的服务。

#### 配置Ingress处理TLS传输

要使控制器能处理TLS请求，我们需要把证书保存要Secret中

```bash
openssl genrsa -out server.key 2048
openssl req -new -x509 -key server.key -out server.crt -days 3650 -subj /CN=http-whoami.example.com

kubectl create secret tls http-whoami-tls --cert=server.crt --key=server.key
```

然后在Ingress中引用它

```yaml
kind: Ingress
apiVersion: networking.k8s.io/v1
metadata:
  name: http-whoami
  labels:
    kubernetes.io/ingress.class: nginx
spec:
  tls:
    - hosts:
        - http-whoami.example.com
      secretName: http-whoami-tls
  rules:
  - host: http-whoami.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: http-whoami
            port:
              name: http
```

最后我们可以通过curl去验证

```bash
curl -vvv -kH 'Host: http-whoami.example.com' https://localhost
```



### 使用就绪探针

只要创建了适当的Pod，那这个Pod几乎立即会成为服务的一部分，并且请求开始呗代理到这个Pod。如果这个Pod需要长时间来加载配置或者数据，那么用户将会请求失败。

#### 介绍就绪探针

与存活探针类似，**Kubernetes还允许为容器定义就绪探针**。就绪探针会被定期调用以确定Pod是否还可以接收客户端请求。就绪探针的类型与存活探针一样有Exec（执行命令）、HttpGet（执行GET请求）和TCP Socket（打开一个连接）。**如果某个Pod还没有准备就绪，则会从服务中删除该Pod，一旦Pod准备就绪，服务就会重新添加Pod**。

**注意：如果Pod未通过就绪检查，Kubernetes并不会终止和重启启动Pod（存活探针会）。**

#### 向Pod添加就绪探针

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: http-whoami-readiness
spec:
  containers:
    - name: app
      image: laboys/http-whoami
      livenessProbe:
        httpGet:
          port: http
          path: /
      readinessProbe:
        httpGet:
          port: http
          path: /
      ports:
        - name: http
          containerPort: 8080
```

#### 就绪探针的实际作用

就绪探针的返回值决定了应用程序是否已经准备好接受客户端的请求。**应该始终为应用定义就绪探针，即使就绪探针只是向发出一个URL请求。**

对于删除/关闭Pod情况下，如果“**Pod收到关闭信号就让就绪探针返回失败**”这**并不是必须**的，因为**一旦Pod被删除，Kubernetes就会从所有服务中移除这个Pod**。



### 使用Headless服务来发现独立的Pod

