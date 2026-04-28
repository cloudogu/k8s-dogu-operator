# Exposition für v2-Dogus

Dieses Feature steuert, wie der `k8s-dogu-operator` `Exposition`-Custom-Resources für v2-Dogus erstellt.

## Konfiguration

Die Funktion wird über die Umgebungsvariable `EXPOSITION_ENABLED` gesteuert.

- `false` (Standard, wenn die Umgebungsvariable nicht gesetzt ist): es werden keine `Exposition`-CRs für v2-Dogus verwaltet
- `true`: der Operator erstellt und aktualisiert `Exposition`-CRs für v2-Dogus auf Basis ihrer Legacy-Webrouten- und Exposed-Port-Konfiguration

Bei Helm-Installationen wird der Wert über `values.yaml` gesetzt:

```yaml
controllerManager:
  env:
    expositionEnabled: true
```

## Verhalten bei `EXPOSITION_ENABLED=true`

Wenn das Feature aktiviert ist, verwendet der Operator für v2-Dogus nicht mehr den Legacy-Pfad über Service-Annotationen, sondern verwaltet eine `Exposition`-CR mit demselben Namen wie das Dogu. Diese CR wird im normalen Reconcile-Ablauf erzeugt oder aktualisiert.

Die Legacy-Service-Annotationen `k8s-dogu-operator.cloudogu.com/ces-services` und `k8s-dogu-operator.cloudogu.com/ces-exposed-ports` werden in diesem Modus nicht mehr an den Dogu-Service geschrieben.
Die externe Erreichbarkeit eines v2-Dogus wird stattdessen vollständig über die `Exposition`-CR beschrieben.

Für HTTP-Routen liest der Operator weiterhin die bekannten Legacy-Informationen aus Image-Konfiguration und Service-Setup aus. Dazu gehören insbesondere:

- `SERVICE_TAGS=webapp`
- port-spezifische Definitionen wie `SERVICE_<PORT>_TAGS`, `SERVICE_<PORT>_NAME`, `SERVICE_<PORT>_LOCATION`, `SERVICE_<PORT>_PASS` und `SERVICE_<PORT>_REWRITE`
- `SERVICE_ADDITIONAL_SERVICES`

Diese Informationen werden auf `spec.http` der `Exposition`-CR abgebildet. 
Dabei werden `location`, `pass` und vorhandene Legacy-Rewrite-Regeln in die HTTP-Struktur der `Exposition` übersetzt.

Bei Dogus mit mehreren exponierten TCP-Ports gilt eine wichtige Regel: Wenn mindestens ein port-spezifisches `SERVICE_<PORT>_TAGS=webapp` gesetzt ist, haben diese port-spezifischen Markierungen Vorrang vor dem globalen `SERVICE_TAGS=webapp`.
Dadurch werden zusätzliche Nicht-HTTP-Ports nicht versehentlich als Web-Routen behandelt.

Zusätzlich werden Exposed Ports aus der `dogu.json` auf Layer-4-Einträge der `Exposition` abgebildet:

- `spec.tcp`
- `spec.udp`

Da das Legacy-Modell der Exposed Ports für diesen Migrationspfad den Protokolltyp nicht präzise genug trennt, wird aktuell für jeden Exposed Port jeweils ein TCP- und ein UDP-Eintrag erzeugt.

Ein vereinfachtes Beispiel:

Dockerfile
```Dockerfile
ENV SERVICE_8080_TAGS=webapp
ENV SERVICE_8080_NAME=jenkins
EXPOSE 8080 50000
```

dogu.json
```json
{
  "ExposedPorts": [
    {
      "container": 50000,
      "host": 50000,
      "type": "tcp"
    }
  ]
}
```

Daraus entsteht sinngemäß eine `Exposition`-CR wie:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Exposition
metadata:
  name: jenkins
spec:
  http:
    - name: jenkins-8080
      service: jenkins
      port: 8080
      path: /jenkins
  tcp:
    - name: port-50000-50000
      service: jenkins
      port: 50000
      requestedExternalPort: 50000
  udp:
    - name: port-50000-50000
      service: jenkins
      port: 50000
      requestedExternalPort: 50000
```

In diesem Beispiel wird nur Port `8080` als HTTP-Route behandelt, obwohl global mehrere Ports exponiert sind.
Der zusätzliche Port `50000` wird nur auf Layer 4 als TCP- und UDP-Eintrag abgebildet.

Wenn ein v2-Dogu weder Webrouten noch Exposed Ports besitzt, entfernt der Operator eine bereits vorhandene `Exposition`-CR für dieses Dogu wieder. Dadurch verbleiben keine leeren Exposition-Ressourcen im Cluster.

## Verhalten bei `EXPOSITION_ENABLED=false`

Wenn das Feature deaktiviert ist, verhält sich der Operator für v2-Dogus wieder im Legacy-Modus. Das bedeutet, dass keine `Exposition`-CRs neu erstellt oder aktualisiert werden.

Stattdessen verwendet der Operator weiterhin die bisherigen Service-Annotationen:

- `k8s-dogu-operator.cloudogu.com/ces-services`
- `k8s-dogu-operator.cloudogu.com/ces-exposed-ports`

Diese Annotationen bleiben in diesem Modus die relevante Eingabe für die nachgelagerten Komponenten des bisherigen Service-Discovery- und Expositionspfads.

Wichtig ist dabei: Beim Ausschalten des Flags erfolgt keine Rückwärtsmigration. Bereits vorhandene `Exposition`-CRs werden in der normalen Reconciliation weder nachträglich in Legacy-Annotationen zurückübersetzt noch allein durch das Deaktivieren des Flags entfernt.

Das Deaktivieren des Flags bedeutet also nur:

- ab diesem Zeitpunkt werden keine neuen `Exposition`-CRs mehr für v2-Dogus erzeugt oder aktualisiert
- neue oder aktualisierte Dogu-Services erhalten wieder die Legacy-Annotationen

Die Löschung vorhandener `Exposition`-CRs passiert weiterhin nur im regulären Deletion-Flow beim Entfernen eines Dogus.
