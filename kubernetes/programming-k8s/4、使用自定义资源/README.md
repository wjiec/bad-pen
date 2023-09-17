使用自定义资源
---

自定义资源（CustomResourceDefinition）是整个 Kubernetes 生态系统中最核心的扩展机制。



### 服务发现

在幕后，kubectl 使用 API 服务所提供的服务发现信息来找到新的资源。我们可以开启 kubectl 的调试日志来了解 kubectl 是如何做到这一点的：

```shell
kubectl get quickstart
I0917 22:41:25.367462   21112 round_trippers.go:463] GET https://172.16.2.6:6443/api?timeout=32s
I0917 22:41:25.367468   21112 round_trippers.go:469] Request Headers:
I0917 22:41:25.367474   21112 round_trippers.go:473]     Accept: application/json;g=apidiscovery.k8s.io;v=v2beta1;as=APIGroupDiscoveryList,application/json
I0917 22:41:25.367479   21112 round_trippers.go:473]     User-Agent: kubectl/v1.27.2 (darwin/arm64) kubernetes/7f6f68f
I0917 22:41:25.390663   21112 round_trippers.go:574] Response Status: 200 OK in 23 milliseconds
I0917 22:41:25.391414   21112 round_trippers.go:463] GET https://172.16.2.6:6443/apis?timeout=32s
I0917 22:41:25.391423   21112 round_trippers.go:469] Request Headers:
I0917 22:41:25.391430   21112 round_trippers.go:473]     Accept: application/json;g=apidiscovery.k8s.io;v=v2beta1;as=APIGroupDiscoveryList,application/json
I0917 22:41:25.391435   21112 round_trippers.go:473]     User-Agent: kubectl/v1.27.2 (darwin/arm64) kubernetes/7f6f68f
I0917 22:41:25.396266   21112 round_trippers.go:574] Response Status: 200 OK in 4 milliseconds
...
I0917 22:41:25.427144   21112 round_trippers.go:463] GET https://172.16.2.6:6443/apis/example.org/v1alpha1/quickstarts?limit=500
I0917 22:41:25.427151   21112 round_trippers.go:469] Request Headers:
I0917 22:41:25.427156   21112 round_trippers.go:473]     Accept: application/json;as=Table;v=v1;g=meta.k8s.io,application/json;as=Table;v=v1beta1;g=meta.k8s.io,application/json
I0917 22:41:25.427160   21112 round_trippers.go:473]     User-Agent: kubectl/v1.27.2 (darwin/arm64) kubernetes/7f6f68f
I0917 22:41:25.433967   21112 round_trippers.go:574] Response Status: 200 OK in 6 milliseconds
```



### 类型定义

CRD 也是 Kubernetes 中的一种资源，从属于 `apiextension.k8s.io/v1beta1`，如下所示：

```yaml
kind: CustomResourceDefinition
apiVersion: apiextensions.k8s.io/v1
metadata:
  name: quickstarts.example.org
spec:
  group: example.org
  names:
    kind: Quickstart
    listKind: QuickstartList
    plural: quickstarts
    singular: quickstart
  scope: Cluster
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          description: "quickstart for crd"
```



### 自定义资源的高级功能

#### 自定义资源合法性验证

在创建或更新自定义资源时，会由 API 服务器进行合法性验证，该验证基于 CRD 定义中的 openAPIV3Schema 进行的：

```yaml
kind: CustomResourceDefinition
apiVersion: apiextensions.k8s.io/v1
metadata:
  name: greeters.example.org
spec:
  group: example.org
  names:
    kind: Greeters
    plural: greeters
    singular: greeter
  scope: Namespaced
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            kind:
              type: string
            apiVersion:
              type: string
            metadata:
              type: object
            spec:
              type: object
              properties:
                schedule:
                  type: string
                  pattern: "^\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}$"
                message:
                  type: string
              required:
                - schedule
            status:
              type: object
              properties:
                phase:
                  type: string
          required:
            - kind
            - apiVersion
            - metadata
            - spec
```

**如果需要更复杂的验证，可以通过准入 Webhook 来实现**

#### 短名字与类别

与原生资源一样，自定义资源可以使用短名，这可以在 CRD 中通过 `shortNames` 字段来定义：

```yaml
kind: CustomResourceDefinition
apiVersion: apiextensions.k8s.io/v1
metadata:
  name: greeters.example.org
spec:
  group: example.org
  names:
    kind: Greeters
    plural: greeters
    singular: greeter
    shortNames:
      - gr
  scope: Namespaced
```

最常用的类别就是 all，我们在定义 CRD 时，也可以指定 `categories` 字段来让 kubectl 列出所有相关的资源：

```yaml
kind: CustomResourceDefinition
apiVersion: apiextensions.k8s.io/v1
metadata:
  name: greeters.example.org
spec:
  group: example.org
  names:
    kind: Greeters
    plural: greeters
    singular: greeter
    categories:
      - all
```

#### 打印列

kubectl 工具使用服务端定义的逻辑来渲染 kubectl get 的输出结果，这可以通过 `additionalPrinterColumns` 来定义：

```yaml
kind: CustomResourceDefinition
apiVersion: apiextensions.k8s.io/v1
metadata:
  name: greeters.example.org
spec:
  group: example.org
  names:
    kind: Greeters
    plural: greeters
    singular: greeter
    shortNames:
      - gr
    categories:
      - all
  scope: Namespaced
  versions:
    - name: v1alpha1
      served: true
      storage: true
      additionalPrinterColumns:
        - name: Schedule
          type: string
          jsonPath: .spec.schedule
        - name: Phase
          type: string
          jsonPath: .status.phase
```

#### 子资源

