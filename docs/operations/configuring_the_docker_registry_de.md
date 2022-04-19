# Konfigurieren der Docker-Registry

Dieses Dokument beschreibt, wie die erforderliche Docker-Registry an den `k8s-dogu-operator` angeschlossen werden kann.
angeschlossen werden kann.

## Voraussetzungen

* Ein K8s-Cluster muss vorhanden sein. Auf diesen sollte über `kubectl` zugegriffen werden können.

## Docker Registry

Die Docker Registry ist ein Speichersystem für die Images des Dogus. Diese Registry enthält die Images
über alle veröffentlichten Dogus und dient somit als Startpunkt für den Dogu-Operator.

Damit eine Docker Registry angehängt werden kann, muss im K8s-Cluster ein Geheimnis erstellt werden.
erstellt werden. Dieses Geheimnis enthält die Anmeldeinformationen, die für den `k8s-dogu-operator` benötigt werden:

1. Docker-Server
2. E-Mail
3. Benutzername
3. Kennwort

## Docker Registry Secret erstellen

Das Geheimnis, das die Docker-Registry-Daten enthält, muss unter dem Namen "k8s-dogu-operator-docker-registry" erstellt werden. Die
Registry-Daten werden im Secret als Docker-JSON-config-Format verschlüsselt. Ein korrektes Geheimnis kann mit `kubectl` erstellt werden.
wie folgt erstellt werden:

```bash
kubectl --namespace <cesNamespace> create secret docker-registry k8s-dogu-operator-docker-registry \
--docker-server="myregistry.mydomain.com" \
--docker-benutzername="meinbenutzername" \
--docker-email="myemail@test.com" \
--docker-password="meinpassword"
```

Danach kann der "k8s-dogu-operator" wie gewohnt [installiert] werden (installing_operator_into_cluster_de.md).