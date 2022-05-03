# Installationsanleitung für den k8s-dogu-operator

## Voraussetzungen

Vor der Installation des Operators müssen die Login Daten für die Dogu- und Docker-Registry hinterlegt
werden:
1. [Docker-Registry](configuring_the_docker_registry_de.md)
2. [Dogu-Registry](configuring_the_dogu_registry_de.md)

## Installation von GitHub

Die Installation von GitHub erfordert die Installations-YAML, die alle benötigten K8s-Ressourcen enthält.

```
GITHUB_VERSION=0.0.6
kubectl apply -f https://github.com/cloudogu/k8s-dogu-operator/releases/download/v${GITHUB_VERSION}/k8s-dogu-operator_${GITHUB_VERSION}.yaml
```

Der Operator sollte nun erfolgreich im Cluster gestartet sein.

## Installation von lokal generiertem Dogu-Operator

Der Dogu-Operator kann mit folgenden Befehlen lokal gebaut und in den Cluster installiert werden. Dabei wird davon ausgegangen, dass der lokale Cluster mit Vagrant aufgesetzt wurde und das Verzeichnis mit dem Vagrantfile als K8S_CLUSTER_ROOT übergeben wird:

```bash
- export K8S_CLUSTER_ROOT=/home/user/k8scluster
- export OPERATOR_NAMESPACE=ecosystem
- make build
```