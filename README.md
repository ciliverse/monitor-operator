# monitor-operator

[English](README.md) | [中文](README_zh.md)

---

### Overview
monitor-operator is a Kubernetes Operator built to simplify and automate the deployment and management of monitoring stacks in Kubernetes clusters.

### Description
This project provides a declarative approach to managing monitoring components in Kubernetes clusters. Through Custom Resource Definitions (CRDs), users can easily deploy, configure, and maintain monitoring-related services such as Prometheus, Grafana, and other monitoring tools.

## Getting Started

### Prerequisites
- go version v1.24.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/monitor-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/monitor-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/monitor-operator:tag
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/monitor-operator/<tag or branch>/dist/install.yaml
```

### By providing a Helm Chart (Recommended)

We provide a production-ready Helm Chart that offers the same functionality as `make deploy` but with better configuration management and deployment flexibility.

#### Option 1: Install from Ciliverse Charts Repository

**Add the Ciliverse Charts repository:**

```bash
# Add Ciliverse Charts repository
helm repo add ciliverse https://charts.cillian.website

# Update Helm repositories
helm repo update

# Search available Charts
helm search repo ciliverse

# Install monitor-operator
helm install monitor-operator ciliverse/monitor-operator \
  --namespace monitoring \
  --create-namespace
```

#### Option 2: Install from Local Chart

**Quick deployment with default settings:**

```sh
helm install monitor-operator ./monitor-operator \
  --namespace monitor-operator-system \
  --create-namespace
```

**Deploy with production configuration:**

```sh
helm install monitor-operator ./monitor-operator \
  -f ./monitor-operator/values-production.yaml \
  --namespace monitor-system \
  --create-namespace
```

**Key advantages of Helm Chart:**
- Flexible configuration through values files
- Easy upgrades and rollbacks
- Support for different environments (dev/staging/prod)
- Standard Kubernetes package management
- Production-ready security configurations
- Built-in monitoring and health checks

For detailed usage instructions, see [monitor-operator/README.md](monitor-operator/README.md) and [monitor-operator/INSTALL.md](monitor-operator/INSTALL.md).

## Contributing

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

