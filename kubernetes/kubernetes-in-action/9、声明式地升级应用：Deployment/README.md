声明式地升级应用：Deployment
----------------------------------------------

在升级应用程序时，我们需要以零停机的方式完成整个升级过程。



### 更新运行在Pod内的应用程序

由于Pod在创建之后不允许直接修改镜像，只能删除原有Pod并使用新的镜像创建Pod镜像替代。有两种方式可以更新所有的Pod

* 直接删除当前现有的所有Pod，然后创建新的Pod：**将会导致应用程序在一段时间内不可用（更新ReplicationController中的image字段）**
* 创建新的Pod并成功运行之后，再删除旧的Pod：**应用程序需要支持两个版本同时对外进行服务，且需要更多的硬件资源**

#### 从旧版本立即切换到新版本

当我们使用第二种方式（先创建新的Pod，再删除旧的Pod）时，Service最初只会将流量转发到初始版本的Pod中，一旦新版本的Pod被创建且正常运行之后，就可以修改服务的表情选择器将流量切到新的Pod上（可以使用`kubectl set selector`命令修改Service的选择器）。



### 使用ReplicationController实现自动的滚动升级（已废弃）

我们还可以执行滚动升级来逐步替代原有的Pod，而不是同时创建所有新的Pod同时删除所有旧的Pod。手动执行这个过程非常的繁琐还容易出错，而kubectl可以自动执行升级过程使得升级更为容易。

在已经有一个v1版本的应用程序时，可以使用以下命令执行滚动升级（**已废弃**）

```bash
kubectl rolling-update app-v1 app-v2 --image=http-whoami:v2
```

这个命令将会有kubectl自动执行以下命令：

1. 修改v1版本的Pod的标签，增加一个`development=xxx`的标签（一共有`app=http-whoami,development=xxx`）
2. 修改v1版本的Rc控制器，为其增加`development=xxx`的选择器
3. 复制一个v1版本的Rc控制命名为app-v2，并将其选择器修改为`development=yyy`，**副本数设置为0**
4. 将新的Rc控制器app-v2的副本数增加1，然后将旧Rc控制器app-v1的副本数量减少1
5. 等待新版本的Pod启动并运行后，重复执行<操作4>直到旧Rc控制器的副本数为0

#### 自动滚动升级能生效的原因

由于Service是通过选择器`app=http-whoami`来选择后端Pod的，而过程中的Rc控制器（不管是app-v1还是app-v2）创建的Pod都带有标签`app=http-whoami`。所以在滚动升级过程中不管是v1还是v2版本的Pod都可以收到请求。

#### 为什么`kubectl rolling-update`被废弃了

首先这个过程会直接修改创建的对象，更重要的是，这个操作是由`kubectl`客户端控制的！这意味着一旦我们在使用`kubectl`滚动升级过程中失去了网络连接将导致升级失败，Pod和Rc控制器最终会处于一个中间状态。



### 使用Deployment声明式地升级应用

Deployment是一种更高阶的资源，用于部署应用程序并以声明的方式升级应用。在使用Deployment时，实际的Pod是由Deployment的Replicaset创建和管理的。使用Deployment可以更容易地更新应用程序，因为可以直接定义单个Deployment资源所需达到的状态，并让Kubernetes处理中间状态。

#### 创建一个Deployment

Deployment与Rc或者Rs没有大的区别，也是由标签选择器、期望副本数和Pod模板组成，Deployment还包含一个额外的部署策略用于定义在修改Deployment时如何执行更新操作。

```yaml
kind: Deployment
apiVersion: apps/v1
metadata:
  name: http-whoami
spec:
  selector:
    matchLabels:
      app: http-whoami
  template:
    metadata:
      labels:
        app: http-whoami
    spec:
      containers:
        - name: app
          image: laboys/http-whoami:v1
  strategy:
    type: RollingUpdate
```

**注意：`--record`选项已经被废弃了，参考[Deprecate and remove --record flag from kubectl](https://github.com/kubernetes/kubernetes/issues/40422)**

##### 展示Deployment滚动过程中的状态

我们可以直接使用`kubectl get deployment`和`kubectl describe deployment`命令来查看Deployment的详细信息。除此之外，我们还有一个命令专门用来查看部署状态

```bash
kubectl rollout status deployment http-whoami
```

##### Deployment管理的Replicaset命名规则

由Deployment创建的Pod名称中均包含一个额外的字母数字（`pod/http-whoami-585fb56fb4-cg52r`），这个实际上对应Deployment和Replicaset中的Pod模板的哈希值（这个哈希值同时也会在Replicaset中出现）。

Deployment会创建多个Replicaset用于对应和管理每个不同版本的Pod模板。

#### 升级Deployment

只需要修改Deployment资源中定义的Pod模板，Kubernetes就会自动将实际的系统状态收敛为资源中定义的状态。

##### 不同的Deployment升级策略

Deployment有两种方式可以达到性的系统状态的方式，这是由Deployment中的升级策略决定的。

* `RollingUpdate`（默认）：执行滚动更新，会渐进式地删除旧的Pod，同时创建新的Pod，使应用程序在整个升级过程中都处于可用状态（**要求应用程序支持多个不同版本共存**）
* `Recreate`：删除旧的Pod之后再开始创建新的Pod，会导致应用程序出现短暂的不可用（**应用程序不支持多个版本同时对外服务时可使用这种策略**）

##### 升级Deployment的方式

修改Deployment等资源有几种不同的方式

* `kubectl edit`：使用默认编辑器打开资源配置，保存退出后将会被更新
* `kubectl patch`：修改单个或者少量资源的属性时非常有用
* `kubectl apply`：通过一个完整的YAML或JSON文件来修改对象属性，如果对象不存在将会自动创建
* `kubectl replace`：将原有对象替换为YAML或JSON文件中定义的新对象，对象必须存在，否则将会报错
* `kubectl set image`：直接修改Pod、Rc、Rs、DaemonSet、Deployment、Job内的镜像

##### Deployment的优点

通过更改Deployment资源的Pod模板，应用程序就已经被升级为一个更新的版本。这个升级过程由运行在Kubernetes上的一个控制器处理和完成，而不是有运行`kubectl rolling-update`的客户端执行。Kubernetes控制器的接管使得整个升级过程变得更加简单可靠。

##### Deployment的升级过程

Deployment背后完成的整个升级过程和执行`kubectl rolling-update`命令非常相似，一个新的Replicaset会被创建然后慢慢扩容，同时之前的Replicaset会慢慢缩容到0（但是最终不会被删除，用于版本的回退）

#### 回滚Deployment

Deployment可以通过以下命令非常容易地回滚到先前部署的版本

```bash
kubectl rollout undo deployment http-whoami
```

同时，`undo`命令也可以在滚动升级过程中运行，并直接停止滚动升级。也可以通过以下命令指定某一个版本号进行回滚

```bash
kubectl rollout undo deployment http-whoami --to-revision=xxx
```

版本号我们可以通过如下命令来获取

```bash
kubectl rollout history deployment http-whoami
```

旧版本的Replicaset过多会导致Replicaset列表过于混乱，我们可以通过指定Deployment的`spec.revisionHistoryLimit`来限制版本历史的数量。

```yaml
spec:
  revisionHistoryLimit: 2
```

#### 控制滚动升级的速率

在Deployment的滚动升级过程中，有2个数学会决定一次替换多少个Pod：

* `spec.strategy.rollingUpdate.maxSurge`：最多允许超出期望副本数的百分比，或者显式的配置个数
* `spec.strategy.rollingUpdate.maxUnavailable`：最多允许有多少原始副本被删除处于可不用状态（支持百分比或者显式配置个数）

比如期望副本数为4，`maxSurge`和`maxUnavailable`都配置为`25%`，则最多可以有`4 + (4 * 25%) = 5`个Pod在运行，最多一次删除`4 * 25% = 1`个原始的Pod。

#### 暂停/恢复滚动升级

我们可以通过以下命令暂停和继续滚动升级

```bash
kubectl rollout pause deployment http-whoami
kubectl rollout resume deployment http-whoami
```

**注意：在暂停过程中，我们可以对Deployment进行多次更改，在更改完成之后再恢复滚动升级**

#### 阻止出错版本的滚动升级

我们可以在`spec.minReadySeconds`属性中指定需要Pod保持就绪状态多久之后才能将其视为可用，这有助于避免部署错误版本的应用程序。

**需要注意的是，`minReadySeconds`需要配合就绪探针一起使用，例如将`minReadySeconds`配置10，表示需要在Pod进入就绪状态10秒之后才能将其标记为可用，并开始替换下一轮的滚动升级。**

#### 为滚动升级配置deadline

默认情况下，在10分钟内不能完成滚动升级的话，本次升级将会被视为失败，这将会导致Deployment自动取消本次升级。我们可以在`spec.progressDeadlineSeconds`中指定Deployment滚动升级失败的超时时间。

