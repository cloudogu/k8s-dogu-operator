# Installation instruction for the k8s-dogu-operator

## Prerequisites

Before installing the operator, the login data for the Dogu and Docker registry must be stored be stored:
1. [Docker registry](configuring_the_docker_registry_en.md)
2. [Dogu registry](configuring_the_dogu_registry_en.md)

## Installation from GitHub

The installation from GitHub requires the installation YAML which contains all required K8s resources. 

```
GITHUB_VERSION=0.0.6
kubectl apply -f https://github.com/cloudogu/k8s-dogu-operator/releases/download/v${GITHUB_VERSION}/k8s-dogu-operator_${GITHUB_VERSION}.yaml
```

The operator should now be successfully started in the cluster.

## Installation of locally generated dogu operator

The dogu operator can be built locally and installed in the cluster using the following commands. This assumes that the local cluster has been set up with Vagrant and the directory containing the Vagrantfile is passed as K8S_CLUSTER_ROOT:

```bash
- export K8S_CLUSTER_ROOT=/home/user/k8scluster
- export OPERATOR_NAMESPACE=ecosystem
- make docker-build
- make image-import
- make k8s-generate
- make k8s-deploy
```