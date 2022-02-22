保障集群内节点和网络安全
-------------------------------------

我们可以配置容器是否被允许访问宿主机节点的资源，以及如何配置以保障Pod间通信的网络的安全。



### 在Pod中使用宿主节点的Linux命名空间

Pod中的容器通常在隔离的命名空间中运行，这些命名空间将容器中的进程与其他容器或者宿主机默认命名空间中的进程隔离开。

#### 在Pod中使用宿主节点的网络命名空间

某些Pod（特别是系统Pod）可能需要在宿主节点的默认命名空间中运行，以方便进程可以看到和节点上的资源和设备。我们可以通过将`pod.spec.hostNetwork`设置为true实现。

```yaml
spec:
  containers:
    - name: app
      image: laboys/http-whoami
  hostNetwork: true
```

Kubernetes控制平面组件通过Pod部署时，这些Pod都会使用`hostNetwork`来让它们的行为与不在Pod中运行时相同

#### 绑定宿主节点的端口而不使用宿主的网络命名空间

当我们仅需要在宿主机上监听一个端口时，我们可以不使用宿主机的网络命名空间，而是通过配置`pod.spec.container.ports.hostPort`属性来实现。

```yaml
spec:
  containers:
    - name: app
      image: laboys/http-whoami
      ports:
        - name: http
          containerPort: 8080
          hostPort: 18080
```

当一个连接到达使用`hostPort`监听的端口时会被直接转发到Pod对应的端口上。**需要注意的是，使用hostPort的Pod仅会在Pod所在节点上绑定端口，而NodePort类型的Service会在所有工作节点上绑定端口，并且每个工作节点只能运行一个带有hostPort属性的相同Pod实例（因为两个进程不能同时绑定宿主机上的同一个端口）。**

#### 使用宿主机的PID和IPC命名空间

与`hostNetwork`选项类型，`pod.spec.hostIPC`和`pod.spec.hostPID`允许Pod使用宿主机的IPC、PID命名空间（看到宿主机上的所有进程，或与宿主机进程进行IPC通信）。

```yaml
spec:
  containers:
    - name: app
      image: laboys/http-whoami
  hostPID: true
  hostIPC: true
```



### 配置节点的安全上下文

除了让Pod使用宿主节点的Linux命名空间，还可以在Pod或其所属容器的配置中通过`security-context`选项配置其他与安全相关的特性。在这个选项中，我们可以配置一下内容

* 指定容器运行进程的用户：`securityContext.runAsUser`
* 阻止容器使用root用户运行：`securityContext.runAsNonRoot`
* 使用特权模式运行容器，使其对宿主节点的内核具有完全的访问权限： `securityContext.privileged`
* 添加或禁用某些内核功能，配置细粒度的内核访问权限：`securityContext.capabilities.add/drop`
* 设置SELinux（Security Enhanced Linux，安全增强型Linux）选项，加强对容器的限制：`securityContext.seLinuxOptions`
* 阻止进程写入容器的根文件系统：`securityContext.readOnlyRootFilesystem`（容器级别）
* 容器使用不同用户时运行共享存储卷：`securityContext.fsGroup/supplementalGroups`

这些配置大多数都可以在**某个容器（`pod.spec.container.securityContext`）或者Pod中的所有容器（`pod.spec.securityContext`）**上生效

#### 使用指定用户运行容器

我们可以通过设置`securityContext.runAsUser`来使用一个与镜像中不同的用户（**注意：使用的是用户ID或者用户组ID**）来运行容器

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: run-as-user-222
spec:
  containers:
    - name: app
      image: alpine
      command: ["/bin/sh", "-c", "--", "while :; do sleep 1d; done"]
      securityContext:
        runAsUser: 222
        runAsGroup: 555
---
kind: Pod
apiVersion: v1
metadata:
  name: run-as-user-333
spec:
  containers:
    - name: app1
      image: alpine
      command: [ "/bin/sh", "-c", "--", "while :; do sleep 1d; done" ]
    - name: app2
      image: alpine
      command: [ "/bin/sh", "-c", "--", "while :; do sleep 1d; done" ]
  securityContext:
    runAsUser: 333
    runAsGroup: 555
```

#### 阻止容器以root用户运行

虽然容器与宿主节点基本上是隔离的，使用root用户运行容器中的进程仍然是一种不好的实践（比如容器可以完全控制挂载进来的宿主节点上的目录）

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: run-as-non-root
spec:
  containers:
    - name: app
      image: alpine
      command: ["/bin/sh", "-c", "--", "while :; do sleep 1d; done"]
      securityContext:
        runAsNonRoot: true
        #runAsUser: 222
```

**注意：此Pod会执行报错`Error: container has runAsNonRoot and image will run as root`，虽然这个Pod被成功调度了，但是不允许运行（可以搭配`runAsUser`一起使用来解决）。**

#### 使用特权模式运行Pod

当Pod需要做在宿主机上才能做的事情（比如访问被保护的操作系统设备或使用一些内核功能（如iptables））时，为了获取宿主机内核的完整权限，这个Pod可以通过`securityContext.privileged`以特权模式运行

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: run-as-privileged
spec:
  containers:
    - name: app
      image: alpine
      command: ["/bin/sh", "-c", "--", "while :; do sleep 1d; done"]
      securityContext:
        privileged: true
```

我们可以通过列出`/dev`目录下所有文件的方式来验证在特权模式下Pod是否能访问宿主节点上的所有设备

```bash
k exec run-as-non-root -ti -- ls /dev
k exec run-as-privileged -ti -- ls /dev 
```

#### 为容器单独添加内核功能

Linux内核已经可以通过内核功能支持更细粒度的权限系统。相对于让容器运行在特权模式下给予无限的权限，更安全的做法是只给予它所需要使用的内核功能的权限。在Kubernetes中通过`securityContext.capabailities.add`来配置

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: run-as-capabilities-add
spec:
  containers:
    - name: app
      image: alpine
      command: ["/bin/sh", "-c", "--", "while :; do sleep 1d; done"]
      securityContext:
        capabilities:
          add:
            - SYS_TIME
```

如上我们可以赋予这个Pod修改硬件时间的能力。**所有可用的内核功能可以通过执行`man capabilities`来查看，需要注意的是在Kubernetes中需要省略`CAP_`前缀。**

#### 在容器中禁用内核功能

与添加内核功能类似，我们可以通过`securityContext.capabailities.drop`来关闭某些内核功能。

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: run-as-capabilities-drop
spec:
  containers:
    - name: app
      image: alpine
      command: ["/bin/sh", "-c", "--", "while :; do sleep 1d; done"]
      securityContext:
        capabilities:
          drop:
            - CHOWN # 关闭修改文件所有者的权限
```

#### 阻止对容器根文件系统的写入

当我们需要阻止容器进程对容器根文件系统的写入可以通过配置`securityContext.readOnlyRootFilesystem`来实现

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: run-as-readonly-root-filesystem
spec:
  containers:
    - name: app
      image: alpine
      command: ["/bin/sh", "-c", "--", "while :; do sleep 1d; done"]
      securityContext:
        readOnlyRootFilesystem: true
```

**为了增强安全性，生产环境中容器最好都配置为不允许对根文件系统进行写入**

#### 容器使用不同用户运行时共享存储卷（Pod级别）

当我们在一个Pod的不同容器间共享存储卷时且每个容器所使用都不是root用户并且用户ID都不同时，我们可以为Pod中的所有容器指定`supplemental`（adj.补充的；增补的）组以允许他们无论以哪个用户允许都可以共享文件。这通过`securityContext.fsGroup`和`securityContext.supplementalGroups`来实现

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: run-as-supplemental-share
spec:
  containers:
    - name: apple
      image: alpine
      command: [ "/bin/sh", "-c", "--", "while :; do sleep 1d; done" ]
      securityContext:
        runAsUser: 111
        runAsGroup: 111
        readOnlyRootFilesystem: true
      volumeMounts:
        - name: shared
          mountPath: /data
    - name: banana
      image: alpine
      command: [ "/bin/sh", "-c", "--", "while :; do sleep 1d; done" ]
      securityContext:
        runAsUser: 222 # no group
        readOnlyRootFilesystem: true
      volumeMounts:
        - name: shared
          mountPath: /data
  securityContext:
    fsGroup: 522
    supplementalGroups:
      - 777
      - 888
  volumes:
    - name: shared
      emptyDir:
        medium: Memory
---
kind: Pod
apiVersion: v1
metadata:
  name: run-as-supplemental-unshared
spec:
  containers:
    - name: apple
      image: alpine
      command: [ "/bin/sh", "-c", "--", "while :; do sleep 1d; done" ]
      securityContext:
        runAsUser: 111
        runAsGroup: 111
        readOnlyRootFilesystem: true
      volumeMounts:
        - name: shared
          mountPath: /data
    - name: banana
      image: alpine
      command: [ "/bin/sh", "-c", "--", "while :; do sleep 1d; done" ]
      securityContext:
        runAsUser: 222 # no group
        readOnlyRootFilesystem: true
      volumeMounts:
        - name: shared
          mountPath: /data
  volumes:
    - name: shared
      emptyDir:
        medium: Memory
```

在创建完成之后我们可以执行以下方法进行验证

```bash
$ k exec run-as-supplemental-share --container apple -ti -- id
uid=111 gid=111 groups=522,777,888

$ k exec run-as-supplemental-share --container banana -ti -- id
uid=222 gid=0(root) groups=522,777,888

$ k exec run-as-supplemental-share --container apple -ti -- sh
	$ echo hello > /data/apple
	$ ls -alh /data
	-rw-r--r--    1 111      522            6 Feb 22 14:46 apple

$ k exec run-as-supplemental-share --container banana -ti -- sh
	$ echo world > /data/banana
	$ ls -alh /data
	-rw-r--r--    1 111      522            6 Feb 22 14:46 apple
	-rw-r--r--    1 222      522            6 Feb 22 14:48 banana
```

在未设置`fsGroup`和`supplementalGroups`情况下（**在某些情况下可能无法读取，看具体共享文件系统对权限的实现决定**）

```bash
$ k exec run-as-supplemental-unshared --container apple -ti -- id
uid=111 gid=111

$ k exec run-as-supplemental-unshared --container banana -ti -- id
uid=222 gid=0(root)

$ k exec run-as-supplemental-unshared --container apple -ti -- sh
	$ echo hello > /data/apple
	$ ls -alh /data
	-rw-r--r--    1 111      111            6 Feb 22 14:50 apple

$ k exec run-as-supplemental-unshared --container banana -ti -- sh
	$ echo world > /data/banana
	$ ls -alh /data
	-rw-r--r--    1 111      111            6 Feb 22 14:50 apple
	-rw-r--r--    1 222      root           6 Feb 22 14:51 banana
```

由于，我们可以总结出以下结论：

* `fsGroup`：在创建文件时起作用，会将文件的组改成`fsGroup`所定义
* `supplemental`：定义用户所关联的额外的组（除`fsGroup`之外的组列表，`fsGroup`会自动添加）
