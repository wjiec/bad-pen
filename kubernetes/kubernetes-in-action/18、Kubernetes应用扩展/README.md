Kubernetes应用扩展
------------------------------

在Kubernetes中我们可以自定义API对象，并为这些对象添加控制器。



### 定义自定义API对象

目前Kubernetes用户使用的大多是相对底层通用的对象，随着行业发展，越来越多的高层对象将会不断涌现，有了这些高层对象后开发者将不需要逐一定义Deployment、Service、ConfigMap等资源，我们可以直接使用自定义控件观察这些对象，并在这些高阶对象的基础上创建底层对象。

#### CustomResourceDefinitions资源

开发者只需向Kubernetes API服务器提交一个CRD对象就可以定义性的资源类型，创建的CRD对象应该要在集群中解决实际问题。通常CRD与所有Kubernetes其他资源一样都有一个相关联的控制器（即基于自定义对象实现目标的组件）。可以按照如下YAML文件所示，创建一个CRD对象

```yaml
kind: CustomResourceDefinition
apiVersion: apiextensions.k8s.io/v1
metadata:
  name: websites.example.com
spec:
  scope: Namespaced
  group: example.com
  names:
    kind: Website
    plural: websites
    singular: website
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                gitRepo:
                  type: string
                domain:
                  type: string
```

CRD对象的资源名字必须与`spec.names`中定义的一致，且一个长的资源名字并不意味着我们一定要用`kind: websites.example.com`来定义，我们可以使用CRD中的`spec.names.kind`所定义的短名称。

在创建我们自定义资源时，我们所使用的`apiVersion`是CRD资源中的`<group>/<version>`形式，如下所示

```yaml
kind: Website
apiVersion: example.com/v1
metadata:
  name: hello.example.com
spec:
  domain: hello.example.com
  gitRepo: https://git.example.com/project-x/docs
```

接下来我们就可以使用`kubectl apply -f`或者`kubectl get/describe/delete website <website-name>`来操作这种类型的资源。

#### 使用自定义控制器器管理资源

有了以上CRD对象之后，之后需要构建和部署一个网站控制器，这个控制器通过API服务器监听Website资源，并为每一个Website资源创建服务和相对应的Pod。每次创建新的Website对象时，API服务器都会发送ADDED监听事件给控制器，这时控制器就可以从Website对象中提取网站名称、域名和源码地址等信息，然后我们就可以将这些信息进行组合并创建Deployment、Service等对象。当Website资源被删除时，API服务器会发送DELETED事件到控制器，这时控制器就可以删除之前创建的Deployment、Service对象。

部署控制器最好的办法是在Kubernetes集群内部运行并为其配置ServiceAccount

```yaml
kind: Deployment
apiVersion: apps/v1
metadata:
  name: website-controller
spec:
  selector:
    matchLabels:
      controller: website
  template:
    metadata:
      labels:
        controller: website
    spec:
      containers:
        - name: controller
          image: example.com/website
      serviceAccountName: controller-website
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: controller:website
subjects:
  - kind: ServiceAccount
    name: controller-website
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluser-admin
```

#### 为自定义对象提供自定义API服务器

如果想更好地支持在Kubernetes中添加自定义对象，最好的方式是使用自己定义的API服务器，并让他直接与客户端进行交互。我们可以在自定义的API服务器上进行YAML文件校验、存储数据等。



### 使用Kubernetes服务目录扩展Kubernetes

服务目录是Kubernetes的扩展内容，使用服务目录可以让运行在Kubernetes中的应用程序轻松地使用外部托管的软件产品（可运行在集群内也可通过代理运行在集群之外）。

#### 服务目录资源

服务目录会添加以下四种通用的资源：

* `ClusterServiceBroker`：描述一个可提供服务的外部系统（服务提供商的地址）

* `ClusterServiceClass`：用于描述可供应的服务类型（比如MySQL、PostgreSQL、Redis等，类比StorageClass）
* `ServiceInstance`：表示根据`ClusterServiceClass`所创建的具体服务实例（类比PV）
* `ServiceBinding`：表示将服务实例与哪些资源进行关联绑定（类比PVC）

当我们创建一个`ClusterServiceBroker`并配置外部系统所提供的服务列表URL，集群就会从配置的地址中拉取服务列表并为每个服务创建一个`ClusterServiceClass`资源。每个`ClusterServiceClass`资源都表示一个服务类型（比如MySQL，PostgreSQL等），每个`ClusterServiceClass`都会关联一个或多个服务方案（比如主从、异地多备等）。

#### 提供服务与使用服务

当需要某个服务时，就直接创建一个`ServiceInstance`资源并在其中指定所需要`ClusterServiceClass`的名字和相对应的服务方案（plan），接下来`ClusterServiceBroker`就会收到这个请求并为我们创建一个对应的服务（这个服务可能在集群内也可能在集群外）。

接下来我们可以将`ClusterInstance`与集群内资源进行绑定，比如与一个`Secret`绑定并将访问服务实例所需的内容放到`Secret`中。

#### 解除绑定与取消服务

一旦不需要服务绑定，就可以通过`kubectl delete servicebinding <binding-name>`来删除绑定，此时服务实例会继续运行且可以给服务实例继续绑定。当我们不需要服务实例时，可以通过`kubectl delete serviceinstance <instance-name>`删除一个服务。



### 基于Kubernetes搭建的平台

基于Kubernetes构建的最著名的PaaS系统包括Deis Workflo和Red Hat的OpenShift。

#### OpenShift容器平台

OpenShift提供了一些在Kubernetes中未提供的功能，如用户管理和群组管理，这能让我们在Kubernets之上运行安全的多用户环境。OpenShift最好的特性之一是与包含程序源代码的Git存储库进行了深度绑定，用户可以在OpenShift集群中快速构建和部署应用程序。

#### Helm

Helm是一个Kubernetes包管理器，它由一个`helm`的客户端程序和一个服务端`Tiller`程序组成（新版本已废弃）。Helm应用程序包被称为图表（Chart），它们与配置结合在一起且合并到图表中以创建一个发行版本（Release），这就可以创建一个应用程序的实例了（包括Development、Service、PVC等）
