高级调度
-------------

更多影响Pod调度到哪个节点的操作



### 使用污点和容忍度阻止节点调度到特定节点

节点污点以及Pod的容忍度被用于限制哪些Pod可以被调度到某一个节点（或者说哪些Pod不能被调度到某个节点），只有当Pod容忍某个节点的污点时，这个Pod才能被调度到该节点。

节点选择器和节点亲缘性规则时通过明确在Pod中添加信息来决定一个Pod可以或不可以被调度到哪些节点，而污点则是在不修改已有Pod信息的前提下，通过在节点上添加污点信息，来拒绝Pod在某些节点上的部署。

#### 污点和容忍度

节点的污点包含一个`key`、`value`以及一个`effect`，表现为`key=value:effect`的形式。我们通过`kubectl descript nodes`查询主节点上污点

```plain
# kubectl descript nodes
Taints:                       node-role.kubernetes.io/master:NoSchedule
```

这个污点的`key`是`node-role.kubernetes.io/master`，`value`的值为空，而`effect`为`NoSchedule`。这个污点将阻止Pod调度到这个节点上，除非有Pod能容忍这个污点（通常容忍这个污点的Pod都是系统级别的Pod）。

我们可以通过`kubectl describe pods`来查询Pod对污点的容忍度，比如我们查询`kube-proxy`的容忍度

```plain
# kubectl -n kube-system describe pods
Tolerations:                 op=Exists
                             CriticalAddonsOnly op=Exists
                             node.kubernetes.io/disk-pressure:NoSchedule op=Exists
                             node.kubernetes.io/memory-pressure:NoSchedule op=Exists
                             node.kubernetes.io/network-unavailable:NoSchedule op=Exists
                             node.kubernetes.io/not-ready:NoExecute op=Exists
                             node.kubernetes.io/pid-pressure:NoSchedule op=Exists
                             node.kubernetes.io/unreachable:NoExecute op=Exists
                             node.kubernetes.io/unschedulable:NoSchedule op=Exists
```

#### 污点的效果

每一个污点都可以关联一个效果

* `NoSchedule`：表示如果Pod没有容忍这些污点，则Pod不能被调度到包含这些污点的节点上（调度时起作用）
* `PreferNoSchedule`：是`NoSchedule`的宽松版本，表示尽量阻止Pod被调度到这个节点上。如果没有其他节点可用了，Pod还是可以被调度到这个节点（调度时起作用）
* `NoExecute`：同时影响调度时和Pod运行时，如果在一个节点上添加了`NoExcute`污点，那么在该节点上运行着的Pod如果没有容忍这个污点将会从这个节点移除。

#### 为节点添加自定义污点

可以通过以下命令给一个节点添加或删除污点

```bash
# 添加污点
kubectl taint node <node-name> <key>[=value]:<effect>

# 删除污点
kubectl taint node <node-name> <key>-
#                                   ^ 注意这个
```

#### 为Pod添加污点容忍度

为Pod添加污点容忍度只能通过YAML方式进行编辑，如下所示

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: http-whoami-prod
spec:
  containers:
    - name: app
      image: laboys/http-whoami
  tolerations:
    - key: production
      effect: NoSchedule
      operator: Exists
```

污点容忍度里的`effect`用于匹配污点的`effect`，如果指定则二者需要对应，而不指定表示匹配所有的污点效果。而`operator`则表示如何进行匹配，可选的值有`Exists`（匹配污点的`key`）和`Equal`（匹配污点的`key`和`value`）。

#### 配置节点失效之后的Pod重新调度的最长等待时间

我们可以通过配置Pod的容忍度来实现当Pod所在节点变成`unready`或`unreachable`时，Kubernetes可以等待该Pod被调度到其他节点的最长等待时间。在未指定的情况下，Kubernetes会自动加上这两个容忍度并配置为300秒，我们也可以自行进行配置修改

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: http-whoami-customer-unready-unreachable
spec:
  containers:
    - name: app
      image: laboys/http-whoami
  tolerations:
    - key: node.kubernetes.io/not-ready
      effect: NoExecute
      operator: Exists
      tolerationSeconds: 600
    - key: node.kubernetes.io/unreachable
      effect: NoExecute
      operator: Exists
      tolerationSeconds: 600
```



### 使用节点亲和性将Pod调度到特定节点上

污点可以用来让Pod远离特定的节点，而节点亲和性则允许你通知Kubernetes将Pod调度（或者优先调度）到某些特定的节点上。

#### 节点亲和性和节点选择器

节点选择器的实现比较简单，只有满足指定要求的节点才能被调度，而节点亲和性除了支持选择特定条件的节点之外还可以根据条件选择优先调度的形式，如果因为资源等无法被调度，则会将其调度到其他节点上。

#### 使用节点亲和性

节点亲和性与节点选择器一样，都是通过节点的标签来进行选择的。我们可以在`pod.spec.affinity.nodeAffinity`中进行设置，如下所示

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: required-labels
spec:
  containers:
    - name: app
      image: laboys/http-whoami
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
          - matchExpressions:
            - key: arch
              operator: In
              values:
                - armv7
                - armv8
```

对于以上属性，`requiredDuringScheduling`表示需要在调度过程中生效，而`IgnoredDuringExecution`则表示忽略运行中的Pod（未来可能会支持`RequiredDuringExecution`特性，在节点标签被修改时自动对节点下的所有Pod执行检查并重新调度）。在`nodeSelectorTerms.matchExpressions`中的语法与`ReplicaSet`中的标签选择器类似，`key`指定需要查询的标签名，而`operator`则表示需要执行的操作（可选的有：`In, NotIn, Exists, NotExist, Gt, Lt`）。

#### 调度时优先考虑某些节点

与节点选择器不同，节点亲和性还可以指定调度时优先选择某些节点，这通过`nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution`来指定。

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: preferred-labels
spec:
  containers:
    - name: app
      image: laboys/http-whoami
  affinity:
    nodeAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 80
          preference:
            matchExpressions:
              - key: arch
                operator: In
                values:
                  - arm
        - weight: 20
          preference:
            matchExpressions:
              - key: gpu
                operator: In
                values:
                  - "true"
```

以上表示将优先选择具有以上计算权重最高的节点。但是需要注意调度器除了节点亲和性的优先级函数，还存在其他的优先级函数会导致并不是所有的Pod都调度到某一类节点（比如`SelectorSpreadPriority`函数用于将属于同一个副本控制器的Pod放到不同的节点上以保障服务的可用性）。



### 使用Pod亲和性与非亲和性对Pod进行协同部署

我们除了通过指定Pod与节点之间的亲和性之外，还可以通过指定Pod与Pod之间的亲和性来让Kubernetes将两个Pod部署在它觉得合适的地方，同时确保两个Pod是靠近的。

#### 使用Pod亲和性将多个Pod部署在同一个节点上

Pod亲和性也是靠标签进行匹配和选择的，如下YAML文件所示

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: database
  labels:
    app: database
spec:
  containers:
    - name: app
      image: laboys/http-whoami
---
kind: Pod
apiVersion: v1
metadata:
  name: backend
  labels:
    app: backend
spec:
  containers:
    - name: app
      image: laboys/http-whoiami
  affinity:
    podAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - topologyKey: kubernetes.io/hostname # 节点范围
          labelSelector:
            matchExpressions:
              - key: app
                operator: In
                values:
                  - database
```

以上内容会将`backend`创建在`database`所在节点上（由`topologyKey`指定范围）。当调度器检查到新Pod创建时会首先找出所有匹配`podAffinity`的所有Pod，并找出这些Pod中具有相同`kubernetes.io/hostname`标签的节点进行调度。

**需要注意的是，Pod亲和性是带有双向亲和的属性，如果此时`database`被删除，重新调度后的Pod也会参考亲和性规则并调度到相同的节点上。**

同时与节点亲和性一样，也可以使用`preferredDuringSchedulingIgnoredDuringExecution`来使用优选方案，如果所选节点无法调度新的Pod就会将其调度到其他节点上（需要指定权重）。

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: backend
  labels:
    app: backend
spec:
  containers:
    - name: app
      image: laboys/http-whoiami
  affinity:
    podAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 80
          podAffinityTerm:
            topologyKey: kubernetes.io/hostname
            labelSelector:
              matchLabels:
                app: database
        - weight: 20
          podAffinityTerm:
            topologyKey: kubernetes.io/zone
            labelSelector:
              matchLabels:
                app: database
```



#### topologyKey的作用

`topologyKey`表示**根据哪个标签挑选适合（近）的节点**，更详细点说调度器会将Pod调度到**与亲和Pod具有相同`topologyKey`标签的节点**上。比如我们有以下节点和对应的Pod

* `noed1(gpu=amd)`：`aa(app=ml_py),bb(app=ml_cc)`
* `noed2(gpu=nv)`：`cc(app=ml_py),dd(app=ml_cc)`
* `node3(gpu=nv)`：`db1(app=database),be1(app=backend)`

此时我们调度一个Pod

* `topology=gpu && podAffinity=cc`：该Pod会具有相同【所选`ml3`所在节点`gpu`标签值】的节点node2和node3
* `topology=gpu && podAffinity=aa`：该Pod会被调度到node1上

#### 利用Pod的非亲和性分开调度Pod

在Kubernetes中可以通过`podAntiAffinity`让Pod彼此远离，在具体场景中，如果两组Pod一起运行会影响彼此的性能或者希望同一组的Pod尽可能分散到不同的节点上，我们可以使用Pod的非亲和性调度方式，如下所示

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: websocket
  labels:
    app: websocket
spec:
  containers:
    - name: app
      image: laboys/http-whoami
---
kind: Pod
apiVersion: v1
metadata:
  name: gateway
  labels:
    app: gateway
spec:
  containers:
    - name: app
      image: laboys/http-whoami
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 80
          podAffinityTerm:
            topologyKey: kubernetes.io/hostname
            labelSelector:
              matchLabels:
                app: gateway
        - weight: 20
          podAffinityTerm:
            topologyKey: kubernetes.io/zone
            labelSelector:
              matchLabels:
                app: websocket
```

以上YAML描述的`gateway`会自己与自己在`kubernetes.io/hostname`冲突，所以每个gateway实例会尽可能分布在不同的节点上。同时`gateway`与`websocket`在`kubernetes.io/zone`上尽可能分开部署。
