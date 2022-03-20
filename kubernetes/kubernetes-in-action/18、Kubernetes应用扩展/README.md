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

