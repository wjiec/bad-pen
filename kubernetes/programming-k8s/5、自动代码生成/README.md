自动代码生成
---

`k8s.io/code-generator` 提供了一组可以在外部使用的代码生成器，这些代码生成器可以用来生成自定义资源相关的代码。



### 调用代码生成器

通常来说，所有控制器项目中都会用类似的方法来使用代码生成器。对于构造一个自定义资源的控制器这样的需求，推荐直接使用来自 `k8s.io/code-generator` 仓库的 `kube_codegen.sh` 脚本。

* `deepcopy-gen`：生成对象的深拷贝方法
* `client-gen`：生成强类型的客户端集合
* `informer-gen`：为自定义类型生成 Informer 对象
* `lister-gen`：为自定义资源生成 Lister 对象



### 通过标签控制代码生成器行为

可以通过在 Go 语言源代码中使用标签来控制代码生成时所使用的属性。标签分为两种：

* 全局标签：在 doc.go 文件中 package 行之前的标签
* 局部标签：在类型声明前的局部标签

具体可以参考官方文档：https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/generating-clientset.md

#### 全局标签

常见的全局标签有：

* `+k8s:deepcopy-gen=package,register`：告诉 deepcopy-gen 为这个包中的所有类型生成对应的深拷贝方法，也可以通过局部标签 `+k8s:deepcopy-gen=false|true` 来指定是否需要为某个类型生成深拷贝方法
* `+groupName=example.org`：用来定义 API 组的全名，这样才能配置正确的 HTTP 路径。通过还可以通过 `+groupGoName` 来作为标识符

#### 局部标签

局部标签可以直接写在 API 类型的前面或者它前面的第二个注释块中：

* `+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object`：用于为这些对象生成正确的深拷贝方法。
* `+genclient`：用于告诉 client-gen 需要为这个类型创建一个客户端。对于集群范围的资源（不在任何工作空间中）可以为类型增加 `+genclient:nonNamespaced` 标签来创建集群范围的客户端。

