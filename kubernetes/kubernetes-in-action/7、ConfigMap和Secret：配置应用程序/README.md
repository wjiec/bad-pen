ConfigMap和Secret：配置应用程序
--------------------------------------------------

几乎所有的应用程序都需要配置信息，并且这些信息不应该被嵌入应用本身。在Kubernetes中，我们可以使用之前的`gitRepo`卷作为配置源，通过这个方式可以保持配置的版本化，并且能比较容器地按需回滚配置。以下方法均可被用作配置应用程序：

* 向容器传递命令行参数
* 为每个容器设置自定义环境变量
* 通过特殊类型的卷将配置文件挂载到容器中



### 配置容器化应用程序

在应用程序的初期，除了将配置嵌入应用本身，通常会以命令行参数的形式配置应用。随着配置选项数量的逐渐增多，就会将配置文件化。另一种通用的传递配置选项给容器化应用程序的方法是借助环境变量。

#### 在Docker中定义命令和参数

容器的完整命令由2部分组成：命令与参数。正确的做法是使用`ENTRYPOINT`定义命令，仅仅使用`CMD`指定所需的默认参数。

在Docker中的`ENTRYPOINT`有两种形式：

* shell形式：`ENTRYPOINT python main.py`
* exec形式：`ENTRYPOINT ["python", "main.py"]`

这两者形式的唯一区别在于在容器中的根进程是否是shell（shell形式的根进程为`/bin/sh`，而exec形式的根进程为`python`）。shell进程一般情况下都是多余，通常情况下都是使用exec形式的`ENTRYPOINT`指令。

#### 在Kubernetes中覆盖命令和参数

在Kubernetes中定义容器时，镜像的`ENTRYPOINT`和`CMD`均可以被覆盖，仅需要在容器定义时设置属性`command`和`args`的值（**command和args字段在创建Pod之后无法被修改**）。

```yaml
spec:
  containers:
  - name: app
    image: python
    command: ["python"]
    args: // ["main.go", "hello", "100"]
      - main.go
      - hello
      - "100"
```

少量的参数可以使用`["a", "b"]`的形式，如果参数比较多的情况下可以使用yaml数组的多行形式。



### 为容器设置环境变量

容器化应用通常会使用环境变量作为配置源。我们一样可以在容器定义中指定环境变量

```yaml
spec:
  containers:
  - name: app
    image: mysql
    env:
    - name: MYSQL_ROOT_PASSWORD
      value: "pass.word"
    - name: MYSQL_DEFAULT_DATABASE
      value: "$(DATABASE_PREFIX)_user"
    - name: DATABASE_PREFIX
      value: "banana"
```

**我们可以在环境变量中通过`$(VAR_NAME)`的方式去引用其他环境变量的值**

#### 硬编码环境变量的不足之处

在Pod定义中硬编码环境变量的值意味着需要显式区分生产环境和开发环境的yaml文件。这在多个环境下复用pod是不利的。



### 利用ConfigMap解耦配置

应用配置的关键在于能够在多个环境中区分配置选项，将配置从应用程序源码中分离。Kubernetes允许将配置选项分离到单独的资源对象（ConfigMap）中，ConfigMap本质上就是一个键值对映射，值可以是短字面量，也是可以完整的配置文件。

应用程序无需直接读取ConfigMap甚至不需要知道ConfigMap是否存在，映射的内容可以通过环境变量或者卷文件的形式传递给容器。*应用程序同样也可以通过Kubernetes REST Api按需直接读取ConfigMap的内容*（**这是不推荐的，应用程序应该不依赖Kubernetes**）

不管应用程序是如何使用ConfigMap的，将配置文件保存在独立的资源对象中有助于在不同环境（开发、测试、生产）下拥有相同名字的配置有助于让Pod适应不同的环境。

#### 创建ConfigMap

创建ConfigMap有很多种方式，最简单的我们可以使用`kubectl create configmap`来创建

```bash
# 创建单个映射条目的字面值
kubectl create configmap literal-config --from-literal=sleep-interval=30
# 包含多个映射条目的字面值
kubectl create configmap multi-literal-config --from-literal=sleep-interval=30 --from-literal=request-timeout=3

# 也可以直接保存完整的配置文件（文件名作为键）
kubectl create configmap file-config --from-file=develop-config.yaml
# 或者指定配置文件所对应的键名
kubectl create configmap custom-key-file-config --from-file=develop=develop-config.yaml
# 深知可以直接引入某一个文件夹中的所有文件
kubectl create configmap dir-file-config --from-file=./configs/

# 也可以将以上所有形式进行组合
kubectl create configmap mixin-config --from-literal=sleep-interval=30 --from-file=develop=develop.yaml --from-file=./certs/
```

创建ConfigMap有多种选项可选（**键名必须满足仅包含数字字母、破折号、下划线以及圆点**）：

* 字面值：直接配置为一个短字面值（字符串）
* 文件：通过读取文件的内容作为值，文件名将作为键（可自定义，但是需要满足条件）
* 文件夹：通过读取文件夹下所有**符合键名规则**的文件为多个条目

#### 给容器传递ConfigMap条目作为环境变量

可以将ConfigMap中的条目作为环境变量传递给Pod，我们可以在`pod.spec.contaienrs.env`中使用`valueFrom`代替`value`即可

```yaml
spec:
  containers:
  - name: app
    image: mysql
    env:
    - name: MYSQL_ROOT_PASSWORD
      valueFrom:
        configMapKeyRef:
          name: database-config
          key: mysql-password
```

**注意：如果在Pod定义中引用了不存在的ConfigMap值时，Kubernetes会正常调度Pod并尝试启动所有的容器，但是引用了不存在键值对的*容器*会启动失败，其他容器能正常启动。我们也可以在`configMapKeyRef`中标记这个键值对是可选的`configMapKeyRef.optional = true`。**

我们也可以一次性传递ConfigMap的所有条目作为环境变量，我们可以使用`envFrom`替换`pod.spec.containers.env`将某个ConfigMap中的所有条目暴露作为环境变量

```yaml
spec:
  containers:
    - name: database
      image: mysql
      envFrom:
        - prefix: CONFIG_
          configMapRef:
            name: http-config
```

**注意：ConfigMap中不正确键名的条目将不会被用作环境变量（Kubernetes不会自动转换键名，比如将-（破折号）转换为_（下划线），因为-（破折号）在环境变量中不合法）**

#### 传递ConfigMap条目作为命令行参数

虽然在`pod.spec.containers.args`中无法直接引用ConfigMap的条目作为参数，但是我们可以以先暴露为环境变量，后在args中通过`$(VAR_NAME)`的方式来间接引用

```yaml
spec:
  containers:
    - name: database
      image: mysql
      env:
        - name: PID_FILE
          valueFrom:
            configMapKeyRef:
              name: database-config
              key: mysql-pid-file
      args:
        - --pid
        - "$(PID_FILE)"
```



### 使用configMap卷将条目暴露为文件

环境变量或者命令行参数作为配置值通常适用于变量值较短的场景，如果需要传递较大的配置文件给容器，可以使用**configMap卷**将ConfigMap中的条目映射为一个文件，运行于容器中的进程可以通过读取文件内容获得对应的配置值。

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: configmap-webserver
spec:
  containers:
    - name: web-server
      image: nginx
      volumeMounts:
        - name: nginx-config
          mountPath: /etc/nginx/conf.d
  volumes:
    - name: nginx-config
      configMap:
        name: webserver-config
```

我们也可以通过`pod.spec.volumes.configMap.items`创建仅包含部分条目的configMap卷。

```yaml
volumes:
  - name: nginx-config
    configMap:
      name: webserver-config
	  items:
		- key: gzip.conf
		  path: use-gzip.conf
```

**注意：指定单个条目时需同时设置条目的键名称以及对应的文件名。**

#### 挂载卷中的部分文件或者文件夹

当我们将卷挂载到某个文件夹时，原镜像中对应文件夹里**原本存在的文件将会被隐藏**（Linux挂载文件系统同样如此），使用configMap卷时也会导致这个问题。

我们可以使用`volumeMounts`中提供的`subPath`字段挂载**卷中的部分文件或者文件夹**。使用这种方式可以挂载文件的同时又不影响原有的文件，但是**会带来文件更新的问题**。

```yaml
spec:
  containers:
    - name: web-server
      image: nginx:alpine
      volumeMounts:
        - name: nginx-config
          mountPath: /etc/nginx/conf.d/999-gzip.conf
          subPath: gzip.conf # 这是卷中的文件名
  volumes:
    - name: nginx-config
      configMap:
        name: webserver-config
        defaultMode: 0600
```

**注意：挂载任意一种卷时均可以使用`subPath`属性，通过这个属性可以选择挂载部分卷而不是挂载完整的卷**

#### 为configMap卷中的文件设置权限

configMap卷中所有文件的权限默认为`644(-rw-r--r--)`，我们可以在`pod.spec.volumes.defaultMode`中进行修改

```yaml
volumes:
  - name: nginx-config
    configMap:
      name: webserver-config
	  items:
		- key: gzip.conf
		  path: use-gzip.conf
	defaultMode: 0600
```

#### 更新应用配置且不重启应用程序

使用环境变量或者命令行参数作为配置源的弊端在于无法在进程运行时更新配置。将ConfigMap暴露为卷可以达到配置热更新的效果（不重新创建Pod情况下更新配置，但是需要应用程序支持）。

当ConfigMap被更新之后，卷中引用它的所有文件也会被更新（所有的文件会被一次性的更新，这是Kubernetes通过符号连接实现的【将整个目录直接链接到新配置上】）。**如果挂载的是单个文件而不是完整的卷，ConfigMap被更新之后对应的文件不会被更新。**

***注意：由于ConfigMap卷中文件的更新对于所有运行的实例不是同步的，所以不同的Pod中的文件可能会在短时间内出现行为不一致的情况。***



### 使用Secret给容器传递敏感数据

配置通常会包含一些敏感数据（比如证书和私钥），为了存储与分发此类信息，Kubernetes提供了一种称为Secret的资源对象。Secret结构与ConfigMap类似，都是键值对的映射，使用方式上也与ConfigMap相似：

* 将Secret条目作为环境变量传递给容器
* 将Secret条目暴露为卷中的文件

Kubernetes只将Secret分发到需要访问的Pod所在节点上以保障其安全性，而且Secret只会存储在节点的内存中，永远不会写入物理存储。我们可以同样可以通过命令行`kubectl create secret`来创建或者通过yaml的形式来创建。

```bash
kubectl create secret generic web-https --from-file=server.key --from-file=server.crt --from-literal=domain=hello.example.com
```

对比Secret和ConfigMap，Secret中的内容会以Base64格式编码，而ConfigMap是以纯文本显示，这就让我们需要在处理yaml或json文件时对内容进行编解码操作。**Secret采用Base64编码的原因是它可以涵盖最大不超过1M的二进制数据。**

#### 在Pod中使用Secret

在Pod中使用Secret的方式与ConfigMap类似

```yaml
kind: Secret
apiVersion: v1
metadata:
  name: secret-server-config
stringData:
  token: hello-world
  server.key: server-key...
  server.crt: server-pem...
---
kind: Pod
apiVersion: v1
metadata:
  name: secret-server
spec:
  containers:
    - name: web-server
      image: nginx:alpine
      volumeMounts:
        - name: html
          mountPath: /usr/share/nginx/html
        - name: certs
          mountPath: /etc/nginx/certs
          readOnly: true
    - name: generator
      image: laboys/fortune
      env:
        - name: REQUEST_TOKEN
          valueFrom:
            secretKeyRef:
              name: secret-server-config
              key: token
      volumeMounts:
        - name: html
          mountPath: /var/www
      args:
        - --token
        - "$(REQUEST_TOKEN)"
  volumes:
    - name: html
      emptyDir: {}
    - name: certs
      secret:
        secretName: secret-server-config
        items:
          - key: server.crt
            path: server.crt
          - key: server.key
            path: server.key
```

可以看到使用Secret的方式基本上与使用ConfigMap相同，Secret的独立条目也可以作为环境变量暴露（使用`secretKeyRef`字段）。

#### 镜像拉取Secret

Kubernetes自身在某些时候希望我们能够传递证书给它（比如从某个私有仓库拉取镜像）。我们可以在`pod.spec.containers.imagePullSecrets`字段中引用这个Secret资源。我们可以通过命令行的方式创建一个拉取秘钥

```bash
k create secret docker-registry example-pulls \
	--docker-server harbor.example.com \
	--docker-username admin \
	--docker-password hello-world \
	--docker-email admin@example.com
```

通过`kubectl create secret docker-registry`来创建一个`docker-registry`类型的秘钥，然后我们就可以让Kubernetes从这个私有仓库中拉取镜像

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: pull-private-image
spec:
  imagePullSecrets:
    - name: example-pulls
  containers:
    - name: app
      image: harbor.example.com/project/app-service
```

**我们可以通过添加Secret到ServiceAccount中来让所有的Pod都能自动添加上拉取秘钥。**
