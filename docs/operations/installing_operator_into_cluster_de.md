# Installationsanleitung für den k8s-dogu-operator

## Installation von GitHub

Die Installation von GitHub erfordert die Installations-YAML, die alle benötigten k8s-Ressourcen enthält.

```
GITHUB_VERSION=0.0.6
kubectl apply -f https://github.com/cloudogu/k8s-dogu-operator/releases/download/v${GITHUB_VERSION}/k8s-dogu-operator_${GITHUB_VERSION}.yaml
```

Der Operator sollte nun erfolgreich im Cluster gestartet sein.