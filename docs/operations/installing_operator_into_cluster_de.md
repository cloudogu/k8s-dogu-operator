# Installationsanleitung für den k8s-dogu-operator

## Voraussetzungen

Vor der Installation des Operators müssen die Login-Daten für die Dogu- und Docker-Registry hinterlegt
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

Der Dogu-Operator kann mit folgendem Befehl lokal gebaut und in den Cluster installiert werden:

```bash
- make build
```

## Von k8s-dogu-operator zusätzlich verwendete Images anpassen

Die ConfigMap `k8s-dogu-operator-additional-images` muss vor dem Start des k8s-dogu-operator existieren. Normalerweise sollte dies
kein Problem sein, da k8s-dogu-operator mit einer vorkonfigurierten ConfigMap ausgeliefert wird.

Ein einzelnes Image in der ConfigMap kann wie folgt durch ein anderes ausgetauscht werden:

```bash
kubectl -n ecosystem get cm k8s-dogu-operator-additional-images -o yaml |
  sed -e 's|chownInitImage: busybox:1.36|chownInitImage: yourimage:tag|' |
  kubectl apply -f -
```

Damit die Änderung dieser Configmap angewendet wird, muss `k8s-dogu-operator` neugestartet werden:

```bash
kubectl -n ecosystem delete pods -l app.kubernetes.io/name=k8s-dogu-operator
```

Die aktuelle Liste der zusätzlichen Images und deren Zugriffsschlüssel:

| Schlüssel        | Image-Beschreibung                                                                                               |
|------------------|------------------------------------------------------------------------------------------------------------------|
| `chownInitImage` | init container image zum Ändern der Dateibesitzverhältnisse vor dem Start eines kubectl. Muss `chown` enthalten. |
