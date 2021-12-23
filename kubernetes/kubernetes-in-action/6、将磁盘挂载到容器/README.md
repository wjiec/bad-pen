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

