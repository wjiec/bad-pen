部署有状态的多副本应用：StatefulSet
-----------------------------------------------------

我们无法简单的使用ReplicationController或者ReplicaSet来运行一个有状态的多副本应用程序，因为Rc或者Rs都是根据Pod模板创建Pod，且每个Pod在被替换后都会有新的唯一标识（网络地址、主机名等）。



### 复制有状态Pod

在使用ReplicationController或ReplicaSet时，如果我们在一个Pod模板中关联特定的存储卷声明，那么所有的副本都会共享这个持久卷声明（绑定到相同的持久卷中），这会导致多个Pod共享同一块存储可能导致出现冲突。我们基于目前的控制器可以有以下方法：

* 手动创建多个Pod，每个Pod使用一个独立的持久卷声明：**缺点是需要手动管理Pod且无法应对故障**
* 手动创建多个ReplicaSet，每个Rs设置副本数为1：**可以应对故障了，但是还是需要手动进行扩缩容**
* 应用程序内协商使用同一个存储的不同位置：**可以应对故障和快速扩缩容了，但是存储可能会成为性能瓶颈**

#### Rc、Rs无法为Pod提供稳定的标识

在有状态的分布式应用程序中，一般都会要求管理员在每个应用程序的配置文件中列出其他所有服务的地址（或主机名）。但是Rc或Rs无法每次创建的Pod都带有一个随机的名字（标识）。

一个取巧的做法是为集群中的每个成员都创建一个独立的Service来提供稳定的网络地址。这个做法也不是很完美，因为这无法让每个Pod知道它所对应的服务地址，不能通过服务IP自动注册。



### 了解StatefulSet

StatefulSet控制器所创建的每个实例都是不可替代的个体，且都有稳定的名字和状态。

#### 对比ReplicaSet和StatefulSet

**相同的地方：**

* 都需要指定一个期待的副本数，且Pod也都是根据Pod模板所创建（但是StatefulSet创建出来的每个Pod都不一样）
* 为了替换挂掉的Pod而创建的新的Pod并不一定会调度到相同的节点上

**不同的地方：**

* Rs中的Pod任何时候都可以被一个全新的Pod替换，而StatefulSet中的Pod挂掉后都需要在别的节点上重建
  * **StatefulSet会保证重启一个新的Pod替代挂掉的Pod，且这个Pod会拥有与前一个Pod完全一致的名称和主机名**
* Rs中的Pod名字是随机的，而StatefulSet中的Pod名字是规律（固定）的
  * **StatefulSet创建的每个Pod都有一个从0开始的顺序索引，这个索引会提现在Pod的名称和主机上，同时也会提现在对应的存储卷上**
* Rs可以选择不创建对应的服务，而StatefulSet通常会要求创建一个用来记录每个Pod网络标记的headless服务
  * **我们可以通过`<pod-name>.<statefulset-name>.<namespace>.svc.cluster.local`访问具体的Pod**
  * 我们也可以通过`<statefulset-name>.<namespace>.svc.cluster.local`获取所有的SRV记录

#### 扩缩容StatefulSet

扩容一个StatefulSet会**使用下一个还没用到的顺序索引值**创建一个新的Pod实例。当缩容一个StatefulSet时**会最先删除拥有最大索引**的那个实例，且StatefulSet**在缩容的任何时刻都只会操作一个Pod实例**。

因为缩容过程如果有多个Pod实例下线，则可能导致数据丢失，而线性的缩容可以让应用程序有空将丢失的数据复制到其他节点上。基于这个原则，**StatefulSet在有实例不健康时不会做缩容操作**。

#### 为StatefulSet实例提供稳定的专属存储

Stateful中的持久存储也是稳定且专属于某一确定实例的。在扩容Stateful时，会同时创建对应序号的Pod和持久卷声明（PVC）。但是**在缩容时，只会删除Pod对象，保留下来的持久卷声明**（PVC）将会预留在下一次扩容时自动绑定并使用（也可以手动删除）。

#### StatefulSet的保障

StatefulSet不仅拥有稳定的标识和独立的存储，它的Pod还有其他的一些保障。当Kubernetes无法确定一个Pod具体的状态时，Kubernetes不能创建一个一模一样的Pod（这会导致系统中存在两个一模一样的Pod在运行，且存储也是一样的），所以Kubernetes必须保证StatefulSet中的Pod实例具有at-most-one语义。也就是说**StatefulSet必须在准确知道一个Pod不在运行之后，才会去创建它的替代Pod**。



### 使用StatefulSet

为了部署一个StatefulSet应用程序，我们需要创建多个不同类型的对象

* 存储数据的持久卷（PV，当集群不存在StrageClass时才需要创建）
* 一个必须的Headless Service
* StatefulSet本身

我们可以使用以下方式进行创建

```yaml
kind: Service
apiVersion: v1
metadata:
  name: ss-http-whoami
spec:
  type: ClusterIP
  clusterIP: None
  selector:
    app.kubernetes.io/name: http-whoami
---
kind: StatefulSet
apiVersion: apps/v1
metadata:
  name: http-whoami
spec:
  replicas: 3
  serviceName: ss-http-whoami
  selector:
    matchLabels:
      app.kubernetes.io/name: http-whoami
  template:
    metadata:
      labels:
        app.kubernetes.io/name: http-whoami
    spec:
      containers:
        - name: app
          image: laboys/http-whoami
          volumeMounts:
            - name: data
              mountPath: /data
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        resources:
          requests:
            storage: 1Mi
        accessModes:
          - ReadWriteOnce
```

值的注意的是，我们在StatefulSet的`spec.template`模板中并没有指定具体的`volumes`声明需要挂载的卷。而这里将会由StatefulSet创建指定Pod时自动将`persistentVolumeClaim`卷添加到Pod中。

StatefulSet会在第一个Pod就绪之后才会开始创建第二个Pod，这是因为状态明确的分布式应用对同时有多个实例启动发生竞争的情况非常敏感，所以依次启动每个实例是比较安全可靠的。

#### 使用API服务器提供的代理功能进行测试

当我们需要检查应用程序时，我们可以在不启动额外Pod的情况下通过API服务器提供的代理功能来实现

```plain
<apiServer>:<port>/api/v1/namespaces/<namespace>/pods/<pod>/proxy/<path>
```

因为API服务器的每次请求都需要添加访问令牌，但是我们还有`kubectl proxy`可以给我们提供免授权的访问

```bash
kubectl proxy
curl -vvv http://localhost:8001/api/v1/namespaces/default/pods/http-whoami-0/proxy/
```

当我们使用`kubectl delete pod http-whoami-1`删除一个Pod时，StatefulSet会使用完全一致的Pod来替换被删除的Pod。扩容和缩容的行为与删除Pod再重建没什么区别。**唯一需要注意的是，StatefulSet的缩容只会删除对应的Pod对象（不会删除PVC和PV对象），且是从序号最大的Pod开始删除**。

我们也可以为这些Pod创建一个普通版本的服务来让外部或者集群内部可以访问

```yaml
kind: Service
apiVersion: v1
metadata:
  name: pub-http-whoami
spec:
  type: ClusterIP
  selector:
    app.kubernetes.io/name: http-whoami
```

然后我们也可以通过API服务器提供的代理接口来进行访问

```bash
<apiServer>:<port>/api/v1/namespaces/<namespace>/services/<service>/proxy/<path>
curl -vvv http://localhost:8001/api/v1/namespaces/default/services/pub-http-whoami/proxy/
```

#### 更新StatefulSet

当我们使用例如`kubectl set image`或者`kubectl patch`或者`kubectl edit`来更新StatefulSet中模板的镜像时，StatefulSet会自动进行滚动升级（1.7版本之前StatefulSet的表现更类似于ReplicaSet，只有新Pod创建时用的才是新镜像）。



### 在StatefulSet中发现伙伴节点

分布式应用中很重要的一个需求是应用程序实例能够发现彼此，这样才能找到集群中的其他成员。

#### SRV记录

DNS中的SRV记录用来指向某一个服务的主机名和端口号，Kubernetes通过一个Headless Service创建SRV记录来指向Pod的主机名。我们可以通过`dig`命令进行查询

```bash
$ dig SRV ss-http-whoami.default.svc.cluster.local

```

当一个Pod需要获取在StatefulSet中其他的Pod时，需要做的只是进行一次简单的DNS SRV查询即可。



### StatefulSet如何处理失效节点

StatefulSet要保证不会有两个拥有相同标记和存储的Pod同时运行，当一个节点失效时，StatefulsSet在明确知道一个Pod不再运行之前它不会且不应该创建一个替代的Pod。

当节点上的kubelet无法与Kubernetes API服务器通信（无法汇报本节点和上面的Pod都在正常运行），过一段时间后控制台就会标记该节点为`NotReady`状态，且该节点上的所有Pod状态变为`Unknown`状态。

当一个Pod变成`Unknown`状态后

* 如果在一段时间后节点能正常联通且正常汇报Pod的状态，那这个Pod会重新标记为`Running`状态
* 如果持续几分钟都无法访问，那这个Pod就会自动从节点上被驱逐（主节点控制），驱逐是通过删除Pod资源来实现的
  * **删除Pod时由于无法与节点进行通信，所以Pod会一直卡在`Terminating`状态，实际上并不会被真正删除**

如果我们确定节点以及彻底失效且不会再次访问时，我们可以使用如下命令强制删除Pod

```bash
kubectl delete pod http-whoami-0 --force --grace-period 0
```

执行这个操作之后StatefulSet将在一个新节点上重新创建新的Pod
