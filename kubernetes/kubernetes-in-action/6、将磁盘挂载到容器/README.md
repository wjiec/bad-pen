卷：将磁盘挂载到容器
-------------------------------

Pod类似于逻辑主机，逻辑主机中运行的程序共享CPU、内存、网络接口等资源，但是**Pod中的每个容器都有自己独立的文件系统**（文件系统来自于容器镜像）。

当需要容器之间需要共享某些数据时，可以使用Kubernetes的卷（Volume）来满足这个需求。卷不像Pod这样的一等公民，它们作为Pod的一部分存在，并和Pod共享相同的生命周期。



### 介绍卷

**Kubernetes卷是Pod的一个组成部分**（在Pod的规范中定义），它们不能单独创建或删除。Pod中的让所有容器都可以使用卷，但**必须先将卷挂载到需要访问的容器中**（可以在文件系统的任务位置挂载卷）。

卷要么从外部资源初始化时填充，要么是在卷内挂载现有目录，要么就是一个空目录。这个填充或装入卷的过程是在Pod内容器启动之前完成的。卷被绑定到Pod的生命周期中，卷只有Pod存在时才会存在，但是根据卷类型，即时Pod和卷销毁之后，卷的文件也可能被持久化在某一个地方并不会被销毁。

卷有多种类型可供选择，其中一些是通用的，也有一些是相对于当前常用的存储技术有较大差别的

* `emptyDir`：用于存储临时数据的空目录（在多个容器之间共享文件）
* `hostPath`：将目录从工作节点的文件系统挂载到Pod中
* `gitRepo`：通过检出Git仓库的内容来初始化的卷
* `nfs`：通过NFS协议挂载的共享卷
* `awsElasticBlockStore/azureDisk/gcePersistentDisk`：用于挂载云服务商提供的特定存储类型
* `cephfs/vsphere-volume/cinder`：其他类型的网络存储
* `configMap/secret/downwardAPI`：将Kubernetes的部分资源或集群信息作为挂载对象的特殊卷
* `persistentVolumeClain`：一种使用预置或者动态配置的持久存储卷

单个容器可以同时使用不同类型的多个卷，每个容器也可以选择装载或者不装载卷。



### 通过卷在容器之间共享数据

卷最简单的用法是用法是在一个Pod的多个容器之间共享数据

#### 使用emptyDir卷

最简单的卷类型是`emptyDir`卷。顾名思义，empty卷从一个空目录开始，运行在Pod内的应用程序可以写入它需要的任何文件。因为卷的生命周期和Pod的生命周期相关联，所以当删除Pod时，卷的内容也会丢失。

**emptyDir卷对于在同一个Pod中运行的容器之间共享文件特别有用**

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: volume-emptydir
spec:
  containers:
    - name: web-server
      image: nginx
      volumeMounts:
        - name: html
          mountPath: /usr/share/nginx/html
      ports:
        - name: http
          containerPort: 80
    - name: blabber
      image: laboys/fortune
      volumeMounts:
        - name: html
          mountPath: /var/www
  volumes:
    - name: html
      emptyDir:
        sizeLimit: 16Mi
        #medium: Memory
```

emptyDir卷是在Pod所在节点的磁盘上创建的，因此其性能取决于节点的磁盘性能。但是我们可以通过修改字段`medium = Memory`来让Kubernetes在内存中创建卷。

emptyDir卷是最简单的卷类型，其他类型的卷都是在它基础上构建的（创建空目录之后再用数据填充它）。



### 访问工作节点文件系统上的文件

某些系统级别的Pod（通常是由DaemonSet启动）确实需要读取节点的文件或者使用节点的文件系统来访问节点设备。Kubernetes通过`hostPath`卷来实现这一点。

hostPath卷指向节点文件系统上的特定文件或目录。在同一个节点上运行并在hostPath卷中使用相同主机路径的Pod可以看到相同的文件。

hostPath卷的内容不会随着Pod被删除时被删除，如果删除了一个Pod，并且下一个Pod（前提是在相同的工作节点上）的hostPath卷使用了相同的主机路径的话，新Pod可以看到上一个Pod留下的数据。

#### 创建和查看hostPath卷

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: hostpath-certs
spec:
  containers:
    - name: app
      image: alpine
      volumeMounts:
        - name: certs
          mountPath: /certs
      command: ["/bin/sh", "-c", "--", "while :; do sleep 1d; done"]
  volumes:
    - name: certs
      hostPath:
        path: /etc/ssl/certs
```

大多数情况下会使用hostPath卷来分节点的日志文件、kubeconfig或CA证书。

**注意：当且仅当需要在节点上读取或写入系统文件时才使用hostPath，切勿使用它们来持久化跨Pod的数据**



### 使用持久化存储

当运行在Pod中的应用程序（MySQL，ElasticSearch等）需要将数据保存到磁盘上，并且即使该Pod重新调度到另一个节点上时也要求具有相同的数据可用，因此我们必须将文件存储到某种类型的网络存储中。

#### 通过底层持久化存储使用卷

根据不同的基础设施（Aliyun，AWS，Azure）使用不同类型的卷，比如在Amazon上应该使用awsElasticBlockStore来提供持久化存储，如果在Azure上运行，则可以使用azureFile或者azureDisk。

如果集群是运行在自有的一组服务器上，那么就有大量其他可一直的选项用于在卷内挂载外部存储。比如可以使用NFS共享来作为卷

```yaml
volumes:
  - name: database
    nfs:
      server: 1.2.3.4
      path: /share/path
```

支持的其他选项有iscsi（挂载ISCSI磁盘资源）、glusterfs（挂载GlusterFS）、rdp（适用于RADOS块设备），还有更多的flexVolume、cinder、cephfs、flocker、fc（光纤通道）等等。

**但是，将这种涉及基础设施类型的信息塞到一个Pod设置中，意味着Pod设置与特定的Kubernetes集群有更大的耦合度。这样就不能在另一个Pod中使用相同的配置了。**



### 从底层存储技术解耦Pod

在Kubernetes集群中为了使应用能够正常请求存储资源，同时避免处理基础设施细节，Kubernetes引入了**持久卷**和**持久卷声明**。研发人员无需向他们的Pod中添加特定技术的卷，而是**由集群管理员设置底层存储然后通过Kubernetes API创建持久卷**并注册（在创建持久卷时，管理员可以指定其大小和所支持的访问模式）。

当集群用户需要在Pod中使用持久化存储时，他们需要**先创建持久卷声明**（PersistentVolumeClain，简称PVC），**指定所需要的最低容量要求和访问模式**，然后用户将PVC提交给Kubernets，如果能**找到可匹配的持久卷（PV）那么就将其绑定到持久卷声明（PVC）中**。

持久卷声明（PVC）可以当做Pod中的一个卷使用，其他用户不能使用相同的持久卷（PV），除非像通过删除持久卷声明（PVC）来释放持久卷（PV）

#### 创建持久卷（PV）

可以使用之前设置Pod中Volume的所有方式来创建持久卷（PV）

```yaml
kind: PersistentVolume
apiVersion: v1
metadata:
  name: pv-retain-data
spec:
  capacity:
    storage: 1Gi
  accessModes:
    - ReadWriteOnce
    - ReadOnlyMany
  persistentVolumeReclaimPolicy: Retain
  hostPath:
    path: /tmp/pv-retain
```

在创建PV时需要告诉Kubernetes对应的容量，以及是否可以由单个节点或者多个节点同时读取或写入。管理员还需要告诉Kubernetes如何处理持久卷（PV）在持久卷声明（PVC）被删除之后的处理方式（`persistentVolumeReclaimPolicy`）。

* `Retain`：在**对应的PVC删除之后，不删除卷（`Released`状态），并且无法再次被PVC再次使用**
* `Delete`：**在对应PVC删除是同步删除卷，但是卷内的文件不会被删除**

**注意：持久卷不属于任何命名空间，它和节点一样是集群层面的资源。**

#### 通过持久卷声明来获取持久卷

当我们需要部署一个需要持久化存储的Pod时，我们可以用到之前创建的持久卷，但是我们不能直接在Pod中使用，需要先声明一个。声明一个持久卷和创建一个Pod是相对独立的过程（即使Pod被重新调度，我们也希望通过相同的持久卷声明来确保可用）。

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: database-pvc
spec:
  resources:
    requests:
      storage: 1Gi
  accessModes:
    - ReadWriteOnce
  storageClassName: ""
```

当创建声明（PVC）时，Kubernetes会找到适当的持久卷（PV）并将其绑定到持久卷声明（PVC）上。持久卷（PV）的容量必须足够大以满足声明的需求，并且卷的访问模式必须包含声明中的访问模式。访问模式与其对应的缩写如下

* `ReadWriteOnce(RWO)`：仅允许单个**节点**挂载读写
* `ReadOnlyMany(ROX)`：允许多个**节点**挂载只读
* `ReadWriteMany(RWX)`：允许多个**节点**挂载读写

**注意：`RWO,ROX,RWX`涉及可以同时使用卷的工作节点的数量而并非Pod的数量**

持久卷是集群范围，因此不能在特定的命名空间中创建，但是持久卷声明又只能在特定的命名空间使用，所以持久卷和持久卷声明只能被同一个命名空间内的Pod创建使用。

#### 在Pod中使用持久卷声明

持久卷现在已经可用了，除非先释放掉卷，否则没有人可以声明相同的卷。在Pod中使用持久卷的方式与之前的没有不同

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: pvc-database
spec:
  containers:
    - name: app
      image: mysql
      ports:
        - name: mysql
          containerPort: 3306
          protocol: TCP
      volumeMounts:
        - name: database
          mountPath: /usr/lib/mysql
  volumes:
    - name: database
      persistentVolumeClaim:
        claimName: database-pvc
```

使用这种间接方法从基础设施里获取存储，对于应用程序开发人员来说更简单，因为研发人员不需要关心底层实际使用的存储技术。

#### 回收持久卷

当我们删除一个持久卷声明并再次创建时，我们可以看到持久卷声明并没有成功而是为`Pending`状态，这是因为之前已经使用过这个持久卷（PV）了，所以它可能包含前一个PVC的数据，如果管理员还没来得及清理，那就不应该将这个卷绑定到全新的声明中。

* 手动回收持久卷：我们在`persistentVolumeReclaimPolicy: Retain`声明持久卷（PV）在从声明（PVC）中释放后仍然保留它和数据内容。我们只能通过手动删除并重新创建持久卷。
* 自动回收持久卷：我们可以配置`persistentVolumeReclaimPolicy: Delete`声明持久卷将在PVC被删除时自动删除卷的内容从而可以被再次使用。

需要注意，某些持久卷可能不支持特定的一些回收策略，**在创建自己的持久卷之前，一定要检查卷中所用到的特定底层存储支持什么回收策略**



### 持久卷的动态卷配置

使用持久卷和持久卷声明可以轻松获得持久化存储资源，但这仍需要一个集群管理来支持创建持久卷。幸运的是，Kubernetes还可以通过动态配置持久卷来自动执行这个任务（分配持久卷）。集群管理员可以创建一个持久卷配置，并定义一个或多个StorageClass对象，从而让开发人员选择他们想要的持久卷类型而不仅仅只是创建持久卷。**StorageClass资源并不属于任一命名空间**。

#### 通过StorageClass资源定义可用存储类型

StorageClass资源指定持久卷声明（PVC）请求此StorageClass时应该使用哪个供应程序来提供持久卷

```yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: localpath
provisioner: rancher.io/local-path
reclaimPolicy: Delete
```

创建StorageClass资源后，用户可以在其持久卷声明中按名称应用存储类

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: pvc-sc-database
spec:
  resources:
    requests:
      storage: 100Mi
  accessModes:
    - ReadWriteOnce
  storageClassName: localpath
```

除了在PVC中指定大小和访问模式，持久卷声明还需要指明所使用的存储类。在创建该PVC后，持久卷（PV）由存储类（StorageClass）资源中指定的`provisioner`创建。

**集群管理员可以创建具有不同性能或其他特性的多个存储类，然后研发人员再决定对应每一个声明最适合的存储类。StorageClass的好处在于，PVC是通过名称去引用SC的，因此只要SC的名词在集群中是相同的，那么PVC便可以跨集群移植。**

#### 不指定存储类的动态配置

我们可以通过`kubectl get sc`获取所有的存储类定义，其中会有一个默认的存储类，我们将这个存储类的定义导出

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
  creationTimestamp: "2021-12-22T15:10:37Z"
  name: standard
  resourceVersion: "284"
  uid: d0bf7f27-8f41-4392-96e7-8b9336927be5
provisioner: rancher.io/local-path
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
```

我们可以看到存储类中有一个注解`storageclass.kubernetes.io/is-default-class`，这会使其成为默认的存储类。**如果在PVC中没有明确指明使用哪个存储类，则会将默认存储类用于提供动态持久卷。**

**当我们需要让PVC绑定到手动创建的PV时，我们可以配置PVC使用`storageClassName: ""`表示不使用默认的存储类动态配置选项。**

* **设置为空字符串**：表示使用自定义配置的PV
* **不设置这个字段**：表示使用默认的存储类生成动态PV

