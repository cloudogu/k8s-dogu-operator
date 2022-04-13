# Konfiguration der Dogu Registry

In diesem Dokument wird beschrieben, wie die benötigte Dogu Registry an dem `k8s-dogu-operator` angeschlossen
werden kann.

## Voraussetzungen

* Es muss ein K8s-Cluster vorhanden sein. Dies sollte via `kubectl` angesteuert werden können.

## Dogu Registry

Bei der Dogu Registry handelt es sich um ein Ablagesystem für Dogus. Diese Registry enthält Information über alle
veröffentlichten Dogus und dient somit als Anlaufstelle für den Dogu Operator.

Damit eine benutzerdefinierte Dogu Konfiguration angebunden werden kann, muss ein Secret im K8s-Cluster angelegt
werden. Dieses Secret enthält den Endpunkt und die für das `k8s-dogu-operator` benötigten Login-Daten:

**Beispiel**

Registry-Endpunkt (API V2): https://my-registry.com/api/v2/
Benutzername: myusername
Passwort: mypassword

## Dogu Registry Secret anlegen

Das Secret mit den Dogu-Registry Daten muss unter dem Namen `k8s-dogu-operator-dogu-registry` angelegt werden. Die 
Registry Daten werden als Literale verschlüsselt im Secret hinterlegt. Ein korrektes Secret kann mit `kubectl` 
folgendermaßen angelegt werden:

```bash
kubectl --namespace <cesNamespace> create secret generic k8s-dogu-operator-dogu-registry \
--from-literal=endpoint="https://my-registry.com/api/v2" \
--from-literal=username="myusername" \
--from-literal=password="mypassword"
```

Im Anschluss kann der `k8s-dogu-operator` wie gewohnt [installiert werden](installing_operator_into_cluster_de.md).

## Benutzerdefinierte Dogu Registry überprüfen

Wenn das Secret korrekt angelegt wurde, dann sollte nach dem Start des `k8s-dogu-operators` dieser eine Log-Ausgabe der
Form: 
`using custom dogu registry <endpoint>` ausgeben.