计算资源管理
-------------------

为一个Pod配置资源的预期使用量和最大使用量是Pod定义中的重要组成部分。这有助于确保Pod可以公平地使用Kubernetes集群资源，同时也会影响整个集群Pod的调度。



### 为Pod中的容器申请资源

我们创建一个Pod时，可以指定容器对CPU和内存的资源请求量（`resources.requrest`）和资源限制量（`resources.limits`）。这些属性是针对每个容器单独指定，Pod对资源的请求量和限制量是它所包含的所有容器的请求量和限制量之和。

#### 创建包含资源请求的Pod

我们可以通过在命令行中带上`--requests=‘cpu=100m,memory=32Mi’`的方式创建Pod，也可以通过YAML文件方式

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: container-res
spec:
  containers:
    - name: app
      image: laboys/http-whoami
      resources:
        requests:
          cpu: 100m
          memory: 32Mi
```

以上我们请求了1/10核的CPU和32MB的内存，这表示我们预期这个容器最大会消耗0.1核CPU和32M内存（这里只是预期，实际上是可以超出的，可以理解为**预留资源**）。需要注意CPU和内存申请量上有一些小规定

* CPU：如果是数字N（可以是浮点数）则表示申请N核的资源，如果是XXXm则表示申请XXX / 1000核的资源
* Memory：如果后缀`Ki、Mi、Gi`则表示是以2为幂（1024进位），如果后缀是`k、M、G`等则表示以10为底（1000进位）

#### 资源请求如何影响调度

调度器在调度时并不关注各类资源在当前时刻的使用量（实际使用量），而只关心节点上已经部署的所有Pod申请的资源量之和。调度器会根据节点所剩下的资源是否能够当前Pod的需求而决定是否将Pod调度到这个节点上。

我们可以通过`kubectl describe nodes`来查看节点的资源占用和总量。**需要注意的是，有些资源会为Kubernetes或者系统组件预留。**

##### 调度器如何利于Pod request为其选择最佳节点

在之前我们学习到调度器首先会比节点列表进行过滤，排除那些不满足要求的节点，然后根据预先配置的优先级函数对其他节点进行排序。其中有两个基于资源请求量的排序函数：

* `LeastRequestedPriority`：将Pod调度到拥有更多未分配资源的节点上（更加平衡）
* `MostRequestedPriotity`：将Pod调度到拥有最少未分配资源的节点上（更加紧凑，适合在云服务平台自动伸缩节点时节约成本）

当Pod因为资源不足无法被调度到节点上而进入`Pending`状态时，我们可以通过删除一些Pod让节点资源被释放，当资源满足要求后，Pod将会被调度。

#### CPU requests如何影响CPU的时间分配

CPU requests不仅在调度时起作用，它同时还决定着剩余（未使用）的CPU时间将会按照`requests.cpu`的比例进行分配

> 节点上共有2000m的CPU资源，此时A申请200m，而B请求800m，机器上剩余的1000m在则会按照1:4分配给A和B。

需要注意的是，如果此时有容器正处于空闲状态，则其他的容器是可以占用全部的CPU资源的。只有当所有容器都需要跑满CPU时才会按照比例进行分配时间片。

#### 定义和申请自定义资源

Kubernetes允许用户未节点添加属于自己的自定义资源，同时支持在Pod requests里申请这种资源。一个自定义资源的好例子是机器上的GPU单元数量。



### 限制容器的可用资源

为了防止Pod使用过多的资源而导致节点故障，我们需要防止容器使用超过指定数量的CPU，而且希望限制容器的可使用内存数量。

#### 设置容器可使用资源的限制

Kubernetes允许用户为每个容器指定资源limits（与requests类似）

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: limited-container
spec:
  containers:
    - name: app
      image: laboys/http-whoami
      resources:
        requests:
          cpu: 200m
          memory: 32Mi
        limits:
          cpu: 1000m
          memory: 128Mi
```

与requests不同，资源limits不受节点可分配资源量的约束，即节点上的所有容器的resource limits可以超出100%。

#### 超过limits

对一个容器的CPU使用率进行限制只会导致**进程分不到比限额更多的CPU时间**而已。而当进程尝试分配比限额更多的内存时将会被杀掉（`OOMKilled`），这是因为内存无法被系统压缩，除非应用程序释放占用的内容，否则其他程序无法获得更多的内存。

当容器被杀死之后，Kubernetes会再次尝试将其重启，如果应用程序因为申请过多资源被再次杀死就会进入`CrashLookBackOff`状态，这会让kubelet在每次崩溃之后以10、20、40、80、160、300秒的延迟时间等待重启，并最终收敛于300秒，直到容器可以正常运行或被删除。

**注意：如果不希望容器被杀掉，最重要的一点是不要将内存Limits设置得很小。而且容器有时即使没有超过限制也会因为OOM而被杀死**

#### 容器内的应用如何看待limits

**在容器中我们看到的始终是节点的内存，而不是容器本身的内存。且容器内看到的同样是节点所有的CPU核心**。运行在容器中的应用最好不要依赖从系统获取的CPU数量和内存大小，我们可以使用`DownwardAPI`将CPU限额传递给容器并使用这个值，或者通过cgroup直接读取配置的CPU限制

* `/sys/fs/cgroup/cpu/cpu.cfs_period_us`
* `/sys/fs/cgroup/cpu/cpu.cfs_quota_us`



### 了解Pod Qos等级

Kubernetes将PodPod划分为3种QoS等级：

* `BestEffort`（最低优先级）：分配给那些没有（**所有容器**）设置任何requests和limits的Pod
  * 在最坏的情况下，它们分不到任何CPU时间，同时在其他Pod需要内存时，这些容器会首先被杀死
* `Guaranteed`（最高优先级）：分配给那些（**所有容器**）设置了相同的requests和limits的Pod
  * 只有在系统进程需要内存时才会被杀死
  * 容器未设置requests时将会与limits相同，所以如果容器只设置了limits则会分配到`Guaranteed`等级
* `Burstable`（中间优先级）：除了以上两种之外的所有情况都属于这个等级
  * 在`BestEffort`的Pod都被杀死之后才会轮到`Burstable`的Pod

对于多容器Pod，如果所有容器的QoS等级相同，则这就是Pod的QoS等级。如果有至少一个容器的QoS等级与其他容器的不同，则这个Pod的QoS等级都是`Burstable`。

#### 内存不足时哪个进程会被杀死

当内存不足时，Kubernetes会从低到高的顺序杀死低优先级的Pod。所以`BestEffort`等级的Pod首先会被杀死，其次是`Burstable`的Pod。

当两个Pod具有相同的QoS等级时，系统将会按照`OOM`分数从高到低杀死对应的Pod。进程的OOM分数通过两个参数计算得出：

* 进程已消耗内存占可用内存（`requests.memory`）的**百分比**
* 基于Pod QoS等级和容器内存申请量固定的OOM分数调节因子

简而言之对于相同QoS的Pod，会首先杀死占用**更高比例**内存的Pod（与实际使用量没关系，比如A的requests.memory=10Mi实际使用9Mi，而B的requests。memory=1000Mi实际使用800Mi，也会先杀死A）。



### 为命名空间中的Pod设置的默认的requests和limits

为每个容器设置requests和limits是一个很好的实践，用户可以通过创建一个LimitRange资源来避免配置每个容器

#### LimitRange资源

LimitRange资源在LimitRange准入控制（Admission）插件上被应用，当API服务器收到创建Pod的请求时，LimitRange插件将会对其进行校验或者填充默认值，如果校验失败，则会直接拒绝。LimitRange资源一个很有用的场景就是阻止用户创建大于单个节点资源量的Pod（会一直Pending）。LimitRange应用于同一个命名空间中的每个独立的Pod、容器或者是PVC等其他类型的对象。

LimitRange资源不仅允许用户为每个命名空间指定容器每种资源的最小和最大值，还支持显式指定资源requests的默认值，甚至可以配置limits和requests的资源比例。

```yaml
kind: LimitRange
apiVersion: v1
metadata:
  name: pod-limited
spec:
  limits:
    - type: Pod
      min:
        cpu: 50m
        memory: 32Mi
      max:
        cpu: 2000m
        memory: 1Gi
```

以上YAML文件指定了在`Pod`级别最小和最大的资源限制

```yaml
kind: LimitRange
apiVersion: v1
metadata:
  name: container-limited
spec:
  limits:
    - type: Container
      min:
        cpu: 50m
        memory: 32Mi
      max:
        cpu: 1000m
        memory: 256Mi
```

以上YAML文件指定了在`Container`级别最小和最大的资源限制

```yaml
kind: LimitRange
apiVersion: v1
metadata:
  name: container-default
spec:
  limits:
    - type: Container
      defaultRequest: # requests的默认值
        cpu: 50m
        memory: 128Mi
      default: # limits的默认值
        cpu: 1000m
        memory: 1Gi
      maxLimitRequestRatio: # limits / requests的最大比例
        cpu: "8"
        memory: "16"
```

以上YAML文件指定`Container`的默认requests和limits资源配置

```yaml
kind: LimitRange
apiVersion: v1
metadata:
  name: pvc-limited
spec:
  limits:
    - type: PersistentVolumeClaim
      min:
        storage: 1Gi
      max:
        storage: 512Gi
```

以上YAML文件指定了在`PersistentVolumeClaim`上的最小和最大的存储容量的限制。

我们可以使用多个LimitRange资源来针对性配置不同类型的限制或者默认值，在LimitRange准入插件中，多个LimitRange对象的限制会在检验Pod和PVC时进行合并。

**注意：LimitRange中配置的limits只能应用于单独的Pod或容器。**



### 限制命名空间中的可用资源总量
