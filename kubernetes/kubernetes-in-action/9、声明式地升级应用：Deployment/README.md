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

