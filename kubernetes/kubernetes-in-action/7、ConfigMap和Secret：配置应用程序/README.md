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
