Pod与集群节点的自动伸缩
-------------------------------------

我们可以通过手动修改ReplicationController、ReplicaSet、Deployment、StatefulSet等资源的`replicas`属性的值来手动实现横向伸缩。也可以通过修改容器资源的requests和limits属性来手动纵向扩容。Kubernetes可以通过监控Pod，并在检测到比如CPU使用率或其他监控项增长时自动对它们进行扩容。



### Pod的横向自动伸缩

Pod的横向伸缩时通过控制器管理Pod的副本数量来实现自动伸缩，这是由Horizontal控制器实现的，我们通过创建一个HorizontalPodAutoscaler（HPA）资源来启用和配置Horizontal控制器。这个控制器会周期性地检查Pod指标，并计算满足HPA资源所配置的目标数值所需的副本数量，进而调整目标资源的replicas字段。

#### 自动伸缩步骤

自动伸缩的过程分为三个步骤：

* 获取被伸缩资源对象所管理的所有Pod指标
* 根据Pod指标计算所需要的副本数量
* 更新被伸缩资源的replicas字段

##### 获取Pod监控指标

Autoscaler本身并不负责采集Pod数据，而是从其他来源获取（Metrics Server），这就意味着集群中需要运行监控服务才能实现自动伸缩。

##### 计算所需的Pod数量

一旦Autoscaler获得了它所需的资源的全部数据，他就可以利用这些数据计算出所需的副本数量。如果只有单个指标，则会将所有Pod上的指标加起来后除以HPA中配置的目标值，再向上取整得到Pod数量。当有多个指标时，Autoscaler将会分别计算每个指标，然后取计算出来的最大值作为Pod的数量。

##### 更新被伸缩资源的副本数

Autoscaler最后会通过更新资源的replicas字段来实现对Pod资源的伸缩工作。

#### 基于CPU使用量进行自动伸缩

因为CPU使用量通常都是不稳定的，比较靠谱的做法是将目标CPU使用量设置地远低于100%（不要超过90%），这有助于预留充分的时间和空间给突发的流量洪峰。就Autoscaler而言，只有Pod的资源请求量与指标有关。通过对比Pod的实际CPU使用量与Pod的资源requests来实现自动伸缩（这意味着我们需要为Pod设置resources.requests）。

#### 基于CPU使用量创建HPA

我们可以直接通过`kubectl autoscale`命令行的形式来创建自动伸缩

```bash
kubectl autoscale deployment http-whoami --max 5 --min 1 --cpu-percent 70
```

以上命令会创建一个HPA指定当CPU使用率超过70%就会进行自动伸缩最多5个Pod，最少保留1个Pod。我们也可以通过YAML文件的形式进行声明

```yaml
kind: HorizontalPodAutoscaler
apiVersion: autoscaling/v2beta2
metadata:
  name: http-whoami-cpu-70
spec:
  minReplicas: 1
  maxReplicas: 5
  scaleTargetRef:
    kind: Deployments
    name: http-whoami
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: AverageValue
          averageUtilization: 70
```

接下来我们可以通过对应用发起请求以观察自动伸缩事件。

##### 自动伸缩的最大速率

Autoscaler单次操作至多使副本数翻倍，且两次扩容操作之间的时间间隔也有限制（只有当3分钟之内没有任何伸缩操作时才会触发扩容），而缩容操作频率更低（需要5分钟）

#### 基于内存使用自动伸缩

基于内存的自动伸缩比基于CPU的困难很多，主要原因在于扩容之后原有的Pod无法释放已经占用的内存，这会导致Autoscaler一直扩容直到达到HPA资源上配置的最大Pod数量。基于内存的自动伸缩配置方法与基于CPU的自动伸缩配置方法完全相同。

#### 基于其他自定义指标进行自动伸缩

在`hpa.spec.metrics`资源中我们可以定义多种不同类型的指标，每个指标都可以指定不同的类型

* `Resource`类型：基于一个资源（CPU、Memory）做出自动伸缩决策
* `Pods`类型：用于引用其他任何种类（包括自定义）与Pod直接相关的指标（比如QPS等）
* `Object`类型：用于让Autoscaler基于非Pod相关的指标来进行伸缩（比如Ingress对象的请求延时，数据库的压力等）。使用Object类型时，Autoscaler只会从单个对象中或者数据（其他将会从所属的所有Pod中获取）

#### 确定哪些指标适合用于自动伸缩

并不是所有指标都适合作为自动伸缩，我们需要用于自动伸缩的指标能因为扩容而减少使用量（比如扩容2个，指标使用量就降到一半）。

#### 缩容到0个副本

允许特定的服务可以被压缩到0个副本叫做空载（idling）与解除空载（un-idling），这有助于大幅度提高硬件了利用率（搭配`MostRequestedPriotity`的调度策略使用）。在新的请求到来时，请求会先被阻塞，直到Pod启动才将请求转发到该Pod上。



### Pod的纵向伸缩

Pod的纵向伸缩通过VPA（VerticalPodAutoscaler）资源实现，具体可参考[Vertical Pod Autoscaler](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler)。



### 集群节点的横向伸缩

Kubernetes支持在需要时立即从云服务提供者请求更多的节点，这通过ClusterAutoscaler资源实现。

#### Cluster Autoscaler

Cluster Autoscaler负责在由于节点资源不足而无法调度Pod到已有节点时自动部署新的节点，同时也会在节点长时间使用率低下时自动下线节点。

##### 从云端基础架构请求新节点

如果一个Pod被创建之后，Scheduler无法将其调度到任何一个已有节点，而ClusterAutoscaler发现这个不可调度Pod时就会从云服务商那请求创建一个新节点。

##### 归还节点

当节点利用率不足是，ClusterAutoscaler也会根据情况减少节点的数目，这是通过监控所有节点上请求（resources.requests）的CPU和内存资源来实现的。**只有当ClusterAutoscaler知道节点上运行的Pod能够重新调度到其他节点时这个节点才会被归还**。当一个节点被选中需要下线时，它首先会标记节点不可调度（通过`kubectl cordon`或者`kubectl drain`可以实现），随后将运行在节点上的Pod疏散到其他节点（删除容器并等待Rc，Rs等重新创建）。
