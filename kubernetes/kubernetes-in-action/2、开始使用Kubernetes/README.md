开始使用Kubernetes
-----------------------------

通过创建一个简单的应用，把它打包成容器镜像并在远端的Kubernetes集群（如GAE）或本地单节点集群中运行，这有助于了解整个Kubernetes体系。

### 创建、运行及共享容器镜像

安装Docker：https://docs.docker.com/get-docker/

#### 创建一个简单的Node.js应用

以下代码接受Http请求并打印应用程序所运行的主机名到响应中，这有助于我们之后做负责均衡时做检查

```js
const os = require('os')
const http = require('http')

console.log('hostname server starting ...')

const server = http.createServer((request, response) => {
    console.log(`received request from ${request.connection.remoteAddress}`)

    response.writeHead(200)
    response.end(`You've hit <${os.hostname()}>`)
})
server.listen(8080)
```

然后我们把该Node应用构建为一个镜像，这需要使用到如下Dockerfile文件

```dockerfile
FROM node:current-alpine3.12

ADD index.js /index.js

ENTRYPOINT ["node", "/index.js"]
```

使用如下命令构建和运行镜像

```bash
# build
docker build -t node-hostname .

# execute
docker run -ti -p 8080:8080 node-hostname
```

##### 镜像是如何构建的？

构建过程不是由Docker客户端进行的，而是将整个目录的文件上传到Docker守护进程并在那里执行。所以**不要在构建目录中包含任何不需要的文件，这样会减慢构建的速度**。

##### 镜像分层

镜像并不是一个大的二进制文件，而是由多层（Layer）组成，**这有助更高效的存储和传输**。构建镜像时，**Dockerfile中的每一条单独的指令都会创建一个新层**（Layer）。

### 配置Kubernetes集群

安装kind：https://kind.sigs.k8s.io/docs/user/quick-start/

安装minikube：https://minikube.sigs.k8s.io/docs/start/

安装kubectl：https://kubernetes.io/docs/tasks/tools/

使用如下命令创建一个本地单节点集群：

```bash
kind create cluster [--config=config.yaml] [--name=k8s]
# 检查集群是否正常工作
kubectl cluster-info
# 通过列出集群节点查看集群是否在运行
kubectl get nodes
# 要查看对象的更详细的信息，可以使用kubectl describe命令
kubectl describe node kind-control-plane
```

#### 在Kubernetes上运行第一个应用

部署应用程序最简单的方式是使用`kubectl run`命令，该命令可以创建所有必要的组件而无需JSON或者YAML配置文件。

```bash
kubectl run node-hostname --image=[image-name] --port=8080
# 或者使用如下方式
kubectl create deployment node-hostname --image=`cat images/node-hostname` --port=8080 

kubectl describe pod
```

##### Pod

在Kubernetes中不会直接处理单个容器，这不是Kubernetes的工作，相反，它使用多个共存容器的理念。这组容器就叫做Pod。一个Pod是一组紧密相关的容器，它们总是一起运行在同一个工作节点上以及同一个Linux命名空间中。

**当执行以上`kubectl`命令后，`kubectl`向`ApiServer`服务器发送一个`REST HTTP`请求，这将会在集群中创建一个新的`Pod`对象。然后调度器就会将其调度到某一个工作节点上，当`kubelet`收到调度任务时，就会告知`docker`从镜像仓库拉取指定的镜像并执行这个镜像。**

#### 访问Pod内容

通过创建`LoadBalancer`类型的服务（服务是类似于`Pod`和`Node`的对象），可以通过负载均衡的公共IP访问Pod。

```bash
kubectl expose pod node-hostname --type=LoadBalancer --name=node-hostname-http
# 列出服务列表
kubectl get services
# 执行请求
docker exec -ti kind-control-plane bash -c "/usr/bin/curl -vvv http://[ip]:8080"

# 获取使用如下方式
kubectl expose deployment node-hostname --port=80 --target-port=8080 --type=LoadBalancer --name node-hostname-http
kubectl get deployments

docker exec -ti kind-control-plane bash -c "/usr/bin/curl -vvv http://[ip]"
docker exec -ti kind-control-plane bash -c "/usr/bin/curl -vvv http://[ip]"
docker exec -ti kind-control-plane bash -c "/usr/bin/curl -vvv http://[ip]"
```

#### 系统的逻辑部分

我们前面的例子并没有直接创建和使用容器，因为Kubernetes的基本单位是Pod。我们的服务结构大致是如下样子：

```plain
Request -> (node-hostname-http) -> (node-hostname-1, node-hostname-2, ...) <- (node-hostname)
		   -------------------     ---------------------------------------     ---------------
			     Service							Pods						  Deployment
```

##### 为什么需要服务？

因为Pod可能会因为任何原因而消失，而服务就是为了解决不断变化的Pod IP问题。当一个服务被创建时，它会得到一个静态的IP，在服务的生命周期中这个IP不会改变。这时客户端通过这个固定的IP访问Pod服务（而不是直接连接Pod），服务会确保请求被其中一个Pod接受处理，而不关心Pod实际运行在哪（IP是什么）。

**服务表示一组或者多组提供相同服务的Pod的静态地址。**

#### 水平伸缩应用

使用Kubernetes的一个好处是可以简单地扩展部署。如下命令将Pod的数量扩展到3个

```bash
kubectl scale deployment --replicas=3
```

**在Kubernetes中，我们不需要告诉Kubernetes应该执行什么操作，而是声明性地改变系统的期望状态，并让Kubernetes检查当前的状态是否与期望的状态一致。这是Kubernetes最基本的原则之一。**

#### 查看应用运行在哪个节点上

正常情况下我们并不需要知道应用程序运行在哪个节点上，如果需要，我们可以使用如下任一命令查看：

```bash
$ kubectl get pods -o wide
NAME                             READY   STATUS    RESTARTS   AGE   IP           NODE                 NOMINATED NODE   READINESS GATES
node-hostname-74d465d4bf-5fwxs   1/1     Running   0          22m   10.244.0.8   kind-control-plane   <none>           <none>
node-hostname-74d465d4bf-bpd2t   1/1     Running   0          22m   10.244.0.7   kind-control-plane   <none>           <none>
node-hostname-74d465d4bf-qh72b   1/1     Running   0          24m   10.244.0.6   kind-control-plane   <none>           <none>

$ kubectl describe pod node-hostname-74d465d4bf-5fwxs
Name:         node-hostname-74d465d4bf-5fwxs
Namespace:    default
Priority:     0
Node:         kind-control-plane/172.18.0.2
Start Time:   Mon, 08 Nov 2021 22:41:30 +0800
Labels:       app=node-hostname
              pod-template-hash=74d465d4bf
Annotations:  <none>
Status:       Running
IP:           10.244.0.8
IPs:
  IP:           10.244.0.8
Controlled By:  ReplicaSet/node-hostname-74d465d4bf
...
```

