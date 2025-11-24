# monitor-operator

[English](README.md) | [中文](README_zh.md)

---

### 项目简介
monitor-operator 是一个基于 Kubernetes Operator 模式构建的监控管理工具，用于简化和自动化监控堆栈的部署和管理。

### 项目描述
该项目提供了一个声明式的方式来管理 Kubernetes 集群中的监控组件。通过自定义资源定义（CRD），用户可以轻松地部署、配置和维护监控相关的服务，如 Prometheus、Grafana 等监控工具。

## 快速开始

### 前置条件
- go 版本 v1.24.0+
- docker 版本 17.03+
- kubectl 版本 v1.11.3+
- 访问 Kubernetes v1.11.3+ 集群的权限

### 部署到集群
**构建并推送镜像到 `IMG` 指定的位置：**

```sh
make docker-build docker-push IMG=<some-registry>/monitor-operator:tag
```

**注意：** 此镜像应该发布到您指定的个人镜像仓库中。
需要确保工作环境能够拉取该镜像。
如果上述命令不起作用，请确保您对镜像仓库有适当的权限。

**将 CRD 安装到集群中：**

```sh
make install
```

**使用 `IMG` 指定的镜像将管理器部署到集群：**

```sh
make deploy IMG=<some-registry>/monitor-operator:tag
```

> **注意**：如果遇到 RBAC 错误，您可能需要授予自己集群管理员权限或以管理员身份登录。

**创建解决方案实例**
您可以从 config/sample 应用示例：

```sh
kubectl apply -k config/samples/
```

>**注意**：确保示例具有默认值以进行测试。

### 卸载
**从集群中删除实例（CR）：**

```sh
kubectl delete -k config/samples/
```

**从集群中删除 API（CRD）：**

```sh
make uninstall
```

**从集群中卸载控制器：**

```sh
make undeploy
```

## 项目分发

以下是向用户发布和提供此解决方案的选项。

### 通过提供包含所有 YAML 文件的包

1. 为构建并发布在镜像仓库中的镜像构建安装程序：

```sh
make build-installer IMG=<some-registry>/monitor-operator:tag
```

**注意：** 上述 makefile 目标在 dist 目录中生成一个 'install.yaml' 文件。
此文件包含使用 Kustomize 构建的所有资源，这些资源是在不依赖其他组件的情况下安装此项目所必需的。

2. 使用安装程序

用户只需运行 'kubectl apply -f <YAML BUNDLE 的 URL>' 即可安装项目，例如：

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/monitor-operator/<tag or branch>/dist/install.yaml
```

### 通过提供 Helm Chart（推荐）

我们提供了一个生产就绪的 Helm Chart，它提供与 `make deploy` 相同的功能，但具有更好的配置管理和部署灵活性。

#### 方式一：从 Ciliverse Charts 仓库安装

**添加 Ciliverse Charts 仓库：**

```bash
# 添加 Ciliverse Charts 仓库
helm repo add ciliverse https://charts.cillian.website

# 更新 Helm 仓库
helm repo update

# 搜索可用的 Charts
helm search repo ciliverse

# 安装 monitor-operator
helm install monitor-operator ciliverse/monitor-operator \
  --namespace monitoring \
  --create-namespace
```

#### 方式二：从本地 Chart 安装

**使用默认设置快速部署：**

```sh
helm install monitor-operator ./monitor-operator \
  --namespace monitor-operator-system \
  --create-namespace
```

**使用生产环境配置部署：**

```sh
helm install monitor-operator ./monitor-operator \
  -f ./monitor-operator/values-production.yaml \
  --namespace monitor-system \
  --create-namespace
```

**Helm Chart 的主要优势：**
- 通过 values 文件进行灵活配置
- 轻松升级和回滚
- 支持不同环境（开发/测试/生产）
- 标准的 Kubernetes 包管理
- 生产就绪的安全配置
- 内置监控和健康检查

详细使用说明请参见 [monitor-operator/README.md](monitor-operator/README.md) 和 [monitor-operator/INSTALL.md](monitor-operator/INSTALL.md)。

## 贡献

**注意：** 运行 `make help` 获取所有潜在 `make` 目标的更多信息

更多信息可以通过 [Kubebuilder 文档](https://book.kubebuilder.io/introduction.html) 找到

## 许可证

Copyright 2025.

根据 Apache License, Version 2.0（"许可证"）获得许可；
除非符合许可证，否则您不得使用此文件。
您可以在以下位置获得许可证副本：

    http://www.apache.org/licenses/LICENSE-2.0

除非适用法律要求或书面同意，否则根据许可证分发的软件是按"原样"分发的，
不提供任何明示或暗示的保证或条件。
请参阅许可证以了解许可证下的特定语言管理权限和限制。