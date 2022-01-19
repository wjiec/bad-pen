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

