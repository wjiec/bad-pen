开始使用Kubernetes
-----------------------------

通过创建一个简单的应用，把它打包成容器镜像并在远端的Kubernetes集群（如GAE）或本地单节点集群中运行，这有助于了解整个Kubernetes体系。

### 创建、运行及共享容器镜像

安装Docker：https://docs.docker.com/get-docker/

#### 创建一个简单的Node.js应用

以下代码接受Http请求并打印应用程序所运行的主机名到响应中，这有助于我们之后做负责均衡时做检查

```js
const os = require('os')
const http = require('http')

console.log('hostname server starting ...')

const server = http.createServer((request, response) => {
    console.log(`received request from ${request.connection.remoteAddress}`)

    response.writeHead(200)
    response.end(`You've hit <${os.hostname()}>`)
})
server.listen(8080)
```

然后我们把该Node应用构建为一个镜像，这需要使用到如下Dockerfile文件

```dockerfile
FROM node:current-alpine3.12

ADD index.js /index.js

ENTRYPOINT ["node", "/index.js"]
```

使用如下命令构建和运行镜像

```bash
# build
docker build -t node-hostname .

# execute
docker run -ti -p 8080:8080 node-hostname
```

##### 镜像是如何构建的？

构建过程不是由Docker客户端进行的，而是将整个目录的文件上传到Docker守护进程并在那里执行。所以**不要在构建目录中包含任何不需要的文件，这样会减慢构建的速度**。

##### 镜像分层

镜像并不是一个大的二进制文件，而是由多层（Layer）组成，**这有助更高效的存储和传输**。构建镜像时，**Dockerfile中的每一条单独的指令都会创建一个新层**（Layer）。

### 配置Kubernetes集群

