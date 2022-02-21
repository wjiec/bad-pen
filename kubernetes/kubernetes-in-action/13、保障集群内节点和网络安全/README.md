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

