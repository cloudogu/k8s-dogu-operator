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

The dogu operator can be built locally and installed in the cluster using the following command.

```bash
- make build
```

## Modifying additional images used by k8s-dogu-operator

The ConfigMap `k8s-dogu-operator-additional-images` must exist prior the k8s-dogu-operator start. Usually this should be of no
problem because k8s-dogu-operator comes with a pre-configured ConfigMap.

The ConfigMap can be modified like so:

```bash
kubectl -n ecosystem get cm k8s-dogu-operator-additional-images -o yaml |
  sed -e 's|chownInitImage: busybox:1.36|chownInitImage: yourimage:tag|' |
  kubectl apply -f -
```

The current list of additional images and their access keys:

| key              | image description                                                                         |
|------------------|-------------------------------------------------------------------------------------------|
| `chownInitImage` | init container image to change file ownership before a dogu starts. Must contain `chown`. |
