Kubernetes API服务器的安全防护
-----------------------------------------------

配置ServiceAccount的权限和在集群中其他产品的权限



### 了解认证机制

API服务器可以配置一到多个的认证插件，在API服务器接收到请求后会经过这些插件的认证，如果其中一个插件可以确定是谁在发送这个请求，那么这个插件就可以提取请求中的用户名、用户ID和用户组信息并返回给API服务器，并进入授权流程。

#### 集群中的用户

认证插件会返回已经认证的用户的用户名，在Kubernetes中存在两种连接到API服务器的客户端

* 真实的用户：用户可以被管理在外部系统中（例如SSO单点登录、LDAP等）
* Pod中的应用程序：Pod使用一种称为`Service Account`的机制，这种机制被创建和存储在集群中的ServiceAccounts资源

#### 集群中的用户组

正常用户和ServiceAccount都可以属于一个或者多个组（组可以一次给多个用户赋予权限），Kubernetes系统中内置了一些特殊的组

* `system:unauthenticated`：所有未通过认证的用户
* `system:authenticated`：所有成功通过认证的用户
* `system:serviceaccounts`：所有在系统中的ServiceAccount
* `system:serviceaccounts:<namespace>`：在特定命名空间中的所有ServiceAccount

#### ServiceAccount介绍

ServiceAccount是一种运行在Pod中的应用程序和API服务器进行身份认证的一种方式。每个Pod都会与一个ServiceAccount（**只能使用同一个命名空间中的ServiceAccount**）相关联，它代表了运行在Pod中的应用程序的身份和能拥有什么样的权限。

**Kubernetes通过将不同的ServiceAccount赋予Pod的方式来控制每个Pod可以访问的资源**

#### 创建ServiceAccount

我们可以简单地使用`kubectl create serviceaccount <service-account-name>`来创建一个ServiceAccount，或者通过提交一个YAML文件实现

```yaml
kind: ServiceAccount
apiVersion: v1
metadata:
  name: simple-sa
imagePullSecrets:
  - name: example-harbor-com
secrets:
  - name: simple-sa-token
```

我们可以通过`kubectl describe serviceaccount <sa-name>`或者使用`kubectl get sa <sa-name> -o yaml`来看出ServiceAccount中配置的数据

**如果在ServiceAccount配置了镜像拉取秘钥，那么使用这个ServiceAccount的Pod将会自动添加到所有使用这个ServiceAccount的Pod中**

#### 将ServiceAccount分配给Pod

我们通过在Pod定义中的`spec.serviceAccountName`来将某个ServiceAccount分配给Pod。

```yaml
spec:
  serviceAccountName: simple-sa
  ...: ...
```



### 通过RBAC加强集群安全

RBAC（Role-Based Acess Control）即基于角色的权限控制。RBAC将用户角色作为决定用户能否执行某个动作的关键因素。主体（可以是一个人、一个ServiceAccount或者是一组用户）与一个或多个角色相关联，每个角色被允许在特定的资源上执行特定的动作。

**Kubernetes中的RBAC会阻止未授权的用户查看和修改集群状态，除非你授予默认的ServiceAccount额外的特权，否则默认的ServiceAccount不允许查看和修改集群状态。**

#### RBAC授权插件

Kubernetes API服务器可以配置使用一个授权插件来检查是否允许用户请求执行某个动作（查看、更新、删除等），RBAC这样的授权插件运行在API服务器中，它会**决定一个客户端是否允许在请求的资源上执行某个动作**。RBAC除了可以对**资源类型**应用安全权限之外，还可以应用于**特定的资源实例**（例如一个名为xxx的服务），甚至还可以应用于**非资源URL**，因为并不是所有的路径都映射到一个资源（例如`/healthz`）。

#### RBAC资源

RBAC授权规则通过Kubernetes中的4中资源来进行配置：

* `Role（角色）， ClusterRole（集群角色）`：指定在资源上可以执行哪些动作
* `RoleBinding（角色绑定），ClusterRoleBinding（集群角色绑定）`：指定某个用户、组或ServiceAccount被绑定到某个角色

**需要注意的是：`Role`和`RoleBinding`是在某个命名空间下的资源（但是可以引用集群角色，只不过`RoleBinging`隶属于某个命名空间），而`ClusterRole`和`ClusterRoleBinding`是集群级别的资源。**

#### 使用Role和RoleBinding

Role资源定义了可以在哪些资源上执行哪些操作，我们可以使用以下YAML来创建一个Role资源

```yaml
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: pod-reader
  namespace: foo # 该角色创建在 foo 命名空间
rules:
  - verbs: # 动作
      - list
      - get
    resources:
      - pods # 定义资源必须使用复数形式
    apiGroups:
      - "" # 所属的API组
```

在角色定义中的每个规则都需要为涉及的资源指定`apiGroup`，我们这边直接通过`resources`定义了可以访问所有的Pod资源，但是我们也可以通过`resourceNames`指定只允许访问某些某些特殊的资源。

我们也可以使用如下命令实现与以上YAML文件一样的效果（命令行会自动匹配相对应的`apiGroup`）

```bash
# kubectl create role <role-name> --verb <verb1> --verb <verb2> --resource <resource-name> --namespace <namespace>
kubectl create role pod-reader-cmd --verb list --verb get --resource pods --namespace foo
```

创建好Role之后，我们就可以将角色绑定到ServiceAccount上了，我们可以使用如下命令来实现绑定

```bash
# kubectl create rolebinding <rolebinding-name> --role <role-name> --serviceaccount <namespace:serviceaccount-name>
kubectl create rolebinding foo-read-pod --role pod-reader --serviceaccount foo:default
```

当然也可以使用以下YAML文件来定义RoleBinding

```yaml
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: foo-read-pod
  namespace: foo
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: pod-reader
subjects:
  - kind: ServiceAccount
    namespace: foo
    name: default
  - kind: ServiceAccount
    namespace: bar
    name: default
```

**RoleBinding始终引用单个角色，但是可以将角色绑定到多个主体上（`subjects`），同时可以注意到RoleBinding可以绑定到非本命名空间中的ServiceAccount上（表示这个bar命名空间的Pod可以列出或查看foo命名空间的Pods资源）**

#### 使用ClusterRole和ClusterRoleBinding

Role和RoleBinding都是命名空间的资源，这意味着他们属于某一个命名空间，但是当我们需要允许跨不同命名空间访问资源，或者访问一些不在命名空间中的特定资源（比如`Node, PersistentVolume, Namespace`等），甚至访问一些不表示资源的URL路径（比如`/api/healthz`），常规的Role不能对这些资源进行授权，但是ClusterRole可以。

ClusterRole是一种集群级别的资源，它允许访问没有命名空间的资源或者非资源类型的资源，也可以作为单个命名空间内部绑定的公共角色从而避免在每个命名空间中都需要重新定义相同的角色。我们可以通过命令或者YAML来创建一个ClusterRole资源

```bash
kubectl create clutserrole pv-reader --verb list,get --resource persistentvolumes
kubectl create clutserrole node-reader --verb list,get --resource nodes
```

接着我们就可以创建一个ClusterRoleBinding资源来将角色绑定到主体上（**注意：非资源类型的ClusterRole只能使用ClusterRoleBinding来绑定，不能使用RoleBinding来引用非资源类型的ClusterRole**）

```bash
kubectl create clusterrolebinding foo-read-pv --clusterrole pv-reader --serviceaccount foo:default
kubectl create clusterrolebinding foo-read-node --clusterrole node-reader --serviceaccount foo:default
```

#### 允许访问非资源类型的URL

API服务器也会对外暴露非资源类型的URL，访问这些URL也必须要显式地授予权限。通常我们会通过`system:discovery`这个`ClusterRole`和同名的`ClusterRoleBinding`来完成对已授权用户的许可（如下输出所示）。

```bash
$ kubectl describe clusterrole system:discovery
Name:         system:discovery
Labels:       kubernetes.io/bootstrapping=rbac-defaults
Annotations:  rbac.authorization.kubernetes.io/autoupdate: true
PolicyRule:
  Resources  Non-Resource URLs  Resource Names  Verbs
  ---------  -----------------  --------------  -----
             [/api/*]           []              [get]
             [/api]             []              [get]
             [/apis/*]          []              [get]
             [/apis]            []              [get]
             [/healthz]         []              [get]
             [/livez]           []              [get]
             [/openapi/*]       []              [get]
             [/openapi]         []              [get]
             [/readyz]          []              [get]
             [/version/]        []              [get]
             [/version]         []              [get]

$ kubectl describe clusterrolebinding system:discovery
Name:         system:discovery
Labels:       kubernetes.io/bootstrapping=rbac-defaults
Annotations:  rbac.authorization.kubernetes.io/autoupdate: true
Role:
  Kind:  ClusterRole
  Name:  system:discovery
Subjects:
  Kind   Name                  Namespace
  ----   ----                  ---------
  Group  system:authenticated
```

**注意：如上ClusterRole引用的是URL路径而不是某一种资源，同时verbs字段也只允许设置为在这些URL上的HTTP方法。**

#### 使用ClusterRole来授权访问指定命名空间的资源

`ClusterRole`可以和`RoleBinding`一起来授权访问`RoleBinding`中的资源。

当我们将一个`ClusterRole`与一个`ClusterRoleBinding`绑定授予一个主体`<namespace>:<sa-name>`时，表示允许这个`sa`访问**所有命名空间**下`ClusterRole`中定义的资源。

当我们将一个`ClusterRole`与一个`RoleBinding`绑定授予一个主体`<namespace>:<sa-name>`时，表示允许这个`sa`访问`RoleBinding`**所在命名空间**下`ClusterRole`中定义的资源。

#### Role、RoleBinding、ClusterRole、ClusterRoleBinding组合的总结

* `ClusterRole, ClusterRoleBinding`：**访问集群级别的资源、非资源类型URL、所有命名空间中的资源**
* `ClusterRole, RoleBinding`：**`RoleBinding`所在命名空间中的资源（一般作为公共角色用于快速授予预定义的权限）**
* `Role, RoleBinding`：**`RoleBinding`所在命名空间的具体资源**

#### 默认的ClusterRole和ClusterRoleBinding

Kubernetes中提供了一组默认的ClusterRole和ClusterRoleBinding，这些资源会在每次API服务器启动时更新（防止意外删除和Kubernetes版本更新时的不兼容问题）。我们可以通过`kubectl get clusterrole/clusterrolebinding`来获取所有预定义的资源。最重要和最常用的一般有以下角色

* `view`：只允许访问一个命名空间中的大多数资源，除了`Role`、`RoleBinding`、`Secret`（Secret中可能包含某个SA的Token，所以不允许读取）
* `edit`：允许查看和修改命名空间中的大多数资源，同时可以读取和修改`Secret`，但是仍不允许读取`Role`和`RoleBinding`（防止权限扩散）
* `admin`：允许查看和修改命名空间中的任何资源，除了`ResourceQuota`和命名空间自身（`Namespace`）
  * `admin`与`edit`的区别在于是否能够读取和修改`Role`和`RoleBinding`\
* `cluster-admin`：获得Kubernetes集群的完全控制权限

Kubernetes中还存在其他默认的`ClusterRole`，这些`ClusterRole`都是以`system:`打头并且勇于各种Kubernetes组件中。比如Controller-Manager虽然作为一个Pod在运行，但是其中的每个控制器都会使用单独的`ClusterRole`和`ClusterRoleBinding`运行（以`system:controller:`为前缀）。

在授予权限时，我们应该遵循**最小权限原则**，即只给每个人提供他们工作所需要的权限。一个比较好的想法是为每一组Pod创建特定的ServiceAccount，并把它与一个特定的`Role`和`RoleBinding`绑定。这样能有效减少入侵者获得集群控制权的可能性。
