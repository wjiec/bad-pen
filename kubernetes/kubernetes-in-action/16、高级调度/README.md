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



### 使用节点亲缘性将Pod调度到特定节点上

