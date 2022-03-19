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
```

CRD对象的资源名字必须与`spec.names`中定义的一致，且一个长的资源名字并不意味着我们一定要用`kind: websites.example.com`来定义，我们可以使用CRD中的`spec.names.kind`所定义的短名称。

在创建我们自定义资源时，我们所使用的`apiVersion`是CRD资源中的`<group>/<version>`形式，如下所示

```yaml
kind: Website
apiVersion: example.com/v1
metadata:
  name: hello.example.com
spec:
  gitRepo: https://git.example.com/project-x/docs
  documentRoot: /var/www
```

接下来我们就可以使用`kubectl apply -f`或者`kubectl get/describe/delete website <website-name>`来操作这种类型的资源。
