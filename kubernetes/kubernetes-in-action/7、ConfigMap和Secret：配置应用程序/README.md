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

