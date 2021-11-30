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

