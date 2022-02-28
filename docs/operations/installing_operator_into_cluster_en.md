# Installation instruction for the k8s-dogu-operator

## Installation from GitHub

The installation from GitHub requires the installation YAML which contains all required k8s resources. 

```
GITHUB_VERSION=0.0.6
kubectl apply -f https://github.com/cloudogu/k8s-dogu-operator/releases/download/v${GITHUB_VERSION}/k8s-dogu-operator_${GITHUB_VERSION}.yaml
```

The operator should now be successfully started in the cluster.