# Konfiguration der Docker Registry

In diesem Dokument wird beschrieben, wie die benötigte Docker Registry an dem `k8s-dogu-operator` angeschlossen
werden kann.

## Voraussetzungen

* Es muss ein K8s-Cluster vorhanden sein. Dies sollte via `kubectl` angesteuert werden können.

## Dogu Registry

Bei der Docker Registry handelt es sich um ein Ablagesystem für die Images der Dogus. Diese Registry enthält die Images 
über alle veröffentlichten Dogus und dient somit als Anlaufstelle für den Dogu Operator.

Damit eine Docker Registry angebunden werden kann, muss ein Secret im K8s-Cluster angelegt
werden. Dieses Secret enthält die für den `k8s-dogu-operator` benötigten Login-Daten:

**Beispiel**

Benutzername: mydockerlogin
Passwort: mydockerpassword

## Docker Registry Secret anlegen

Das Secret mit den Docker-Registry Daten muss unter dem Namen `k8s-dogu-operator-docker-registry` angelegt werden. Die 
Registry Daten werden als Literale verschlüsselt im Secret hinterlegt. Ein korrektes Secret kann mit `kubectl` 
folgendermaßen angelegt werden:

```bash
kubectl --namespace <cesNamespace> create secret generic k8s-dogu-operator-docker-registry \
--from-literal=username="myusername" \
--from-literal=password="mypassword"
```

Im Anschluss kann der `k8s-dogu-operator` wie gewohnt [installiert werden](installing_operator_into_cluster_de.md).