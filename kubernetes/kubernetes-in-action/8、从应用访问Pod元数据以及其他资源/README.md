从应用访问Pod元数据以及其他资源
--------------------------------------------------

应用在运行过程中可能需要获取所运行环境的一些信息，包括应用自身以及集群中其他组件的信息。

通过环境变量或者ConfigMap、Secret卷向容器传递配置信息对于Pod调度、运行前预设的数据是可行的，但是对于不能预先知道的数据（比如Pod的IP、主机名、Pod自身的名称等）就无法处理了。



### 通过DownwardAPI传递元数据

Downward API允许我们通过环境变量或者文件（downwardAPI卷中）方式传递Pod的元数据。**需要注意的是：DownwardAPI并不是像REST endpoint那样需要通过访问接口来获取数据**。

DownwardAPI可以给再Pod中运行的进程暴露Pod元数据，目前可以给容器传递以下数据：

* Pod的名称（`fieldPath=metadata.name`）
* Pod的IP（`fieldPath=status.podIP`）（**只能通过环境变量暴露**）
* Pod所在的命名空间（`fieldPath=metadata.namespace`）
* Pod运行节点的名称（`fieldPath=spec.nodeName`）（**只能通过环境变量暴露**）
* Pod运行所属的服务账号的名称（`fieldPath=spec.serviceAccountName`）（**只能通过环境变量暴露**）
* 每个容器请求的CPU和内存的使用量（`resourceFieldRef.resource=requests.cpu`）
* 每个容器可以使用的CPU和内存的限制（`resourceFieldRef.resource=requests.memory`）
* Pod的标签（**只能通过卷暴露**）
* Pod的注解（**只能通过卷暴露**）

#### 通过环境变量暴露元数据

我们可以通过环境变量的方式进行暴露以上部分的元数据

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: downward-env
spec:
  containers:
    - name: printer
      image: busybox
      env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: POD_IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: SERVICE_ACCOUNT
          valueFrom:
            fieldRef:
              fieldPath: spec.serviceAccountName
        - name: CONTAINER_CPU_REQUEST_MILLICORES
          valueFrom:
            resourceFieldRef:
              resource: requests.cpu
              divisor: 1m
        - name: CONTAINER_MEMORY_REQUEST_MIBIBYTES
          valueFrom:
            resourceFieldRef:
              resource: requests.memory
              divisor: 1Mi
        - name: CONTAINER_CPU_LIMIT_MILLICORES
          valueFrom:
            resourceFieldRef:
              resource: limits.cpu
              divisor: 1m
        - name: CONTAINER_MEMORY_LIMIT_MIBIBYTES
          valueFrom:
            resourceFieldRef:
              resource: limits.memory
              divisor: 1Mi
      command: ["/bin/sh", "-c", "--", "while :; do sleep 1d; done"]
      resources:
        requests:
          cpu: 16m
          memory: 16Mi
        limits:
          cpu: 64m
          memory: 64Mi
```

当容器进程运行时就可以通过以上这些环境变量获取所有的这些元数据了。

```bash
$ env # 经过整理和修剪
POD_NAME=downward-env
POD_NAMESPACE=default
POD_IP=10.244.1.14
NODE_NAME=mercury
SERVICE_ACCOUNT=default
CONTAINER_CPU_REQUEST_MILLICORES=16
CONTAINER_MEMORY_REQUEST_MIBIBYTES=16
CONTAINER_CPU_LIMIT_MILLICORES=64
CONTAINER_MEMORY_LIMIT_MIBIBYTES=64
```

其中比如`request.cpu`或者`limits.cpu`中关于CPU的使用量可以使用`1m`来表示千分之一核的计算能力。也可以直接使用`1`来表示需要1个核心。`request.memory`和`limits.memory`中则可以使用`2Ki, 3Mi, 4Gi`等待后缀`i`表示使用的是二进制的”整数“。

#### 使用downwardAPI卷来传递元数据

如果更倾向于使用文件的方式而不是环境变量的方式暴露元数据，可以定义个downwardAPI卷并挂载到容器中

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: downward-file
spec:
  containers:
    - name: printer
      image: alpine
      command: ["/bin/sh", "-c", "--", "while :; do sleep 1d; done"]
      resources:
        requests:
          cpu: 16m
          memory: 16Mi
        limits:
          cpu: 64m
          memory: 64Mi
      volumeMounts:
        - name: downward
          mountPath: /var/downward
  volumes:
    - name: downward
      downwardAPI:
        items:
          - path: pod/name
            fieldRef:
              fieldPath: metadata.name
          - path: pod/namespace
            fieldRef:
              fieldPath: metadata.namespace
          - path: requests/cpu
            resourceFieldRef:
              containerName: printer
              resource: requests.cpu
          - path: requests/memory
            resourceFieldRef:
              containerName: printer
              resource: requests.memory
          - path: limits/cpu
            resourceFieldRef:
              containerName: printer
              resource: limits.cpu
          - path: limits/memory
            resourceFieldRef:
              containerName: printer
              resource: limits.memory
          - path: pod/labels
            fieldRef:
              fieldPath: metadata.labels
          - path: pod/annotations
            fieldRef:
              fieldPath: metadata.annotations
```

在downwardAPI卷中我们通过`items`属性定义需要暴露的元数据。我们可以进入容器看到具体内容

```bash
$ tree /var/downward
.
├── requests
│   ├── memory
│   └── cpu
├── pod
│   ├── namespace
│   ├── name
│   ├── labels
│   └── annotations
└── limits
    ├── memory
    └── cpu
$ cat /var/downward/pod/annotations
kubectl.kubernetes.io/last-applied-configuration="{...}\n"
kubernetes.io/config.seen="2022-01-08T14:12:32.146637591Z"
```

需要注意的是，当暴露容器级别的元数据时（容器可使用的资源限制等）必须指定引用资源字段对应的容器名称（这允许你**向一个容器报告另一个容器的使用量等信息**），如果使用**环境变量方式，则只能传递自身的资源请求和限制元数据**。

#### 通过环境变量或者卷暴露元数据的区别

当我们再Pod运行时修改了标签和注解之后，Kubernetes会更新存有相关信息的文件，从而使Pod可以获取最新的数据。所以使用卷可以获得当前生效的元数据，而通过环境变量的方式则无法获得最新的值，看到的总是当初的快照。



### 与Kubernetes API服务器交互

DownwardAPI仅仅可以暴露一个Pod自身的元数据，而且只可以暴露部分元数据。但我们需要其他Pod的信息，甚至是集群中其他资源的信息时，我们就需要直接与API服务器进行交互了。

#### 在外部访问Kubernetes访问REST API

我们可以在集群内部（Pod之外）通过`kubectl cluster-info`来获取集群的API服务器地址，并通过curl与之进行通信

```bash
$ kubectl cluster-info
$ curl -k https://127.0.0.1:xxxx
{
  "kind": "Status",
  "apiVersion": "v1",
  "metadata": {
    
  },
  "status": "Failure",
  "message": "forbidden: User \"system:anonymous\" cannot get path \"/\"",
  "reason": "Forbidden",
  "details": {
    
  },
  "code": 403
}
```

因为我们没传入访问Token，所以会直接得到403错误，但是我们可以使用`kubectl proxy`命令创建一个代理与服务器进行交互，这个代理将会替我们处理所有的鉴权问题。

```bash
$ kubectl proxy
Starting to serve on 127.0.0.1:8001

$ curl localhost:8001
{
  "paths": [
    "/.well-known/openid-configuration",
    "/api",
    "/api/v1",
    "/apis",
    "/apis/",
	"..."
  ]
}
```

当我们直接访问这个API时。API服务器会返回一组路径，这些路径对应了我们创建Pod、Service等资源时对应的API组（**没有被列入API组的资源类型并不属于任何组，原因是Kubernetes初期并没有API组这个概念，现在一般被认为是核心API组**）和版本信息。

我们可以按照路径列表进行拼接路径来访问特定的资源

```bash
curl localhost:8001/api/v1/pods # 查看所有的Pod
curl localhost:8001/apis/batch/v1/namespace/default/jobs/my-job # 访问指定的单个资源
```

