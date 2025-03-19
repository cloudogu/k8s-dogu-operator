# Export-Modus für Dogus

Für die Migration einer Multinode CES-Instanz müssen die Daten aller Dogus von der Quell-Instanz in die Ziel-Instanz kopiert werden.
Damit die Downtime während der Migration möglichst gering ist, sollen die Daten kopiert werden, während die Quell-Instanz weiterhin im Betrieb ist.
Der "Export-Modus" eines Dogus stellt das Daten-Volume des Dogus über einen "Exporter"-Sidecar-Container für die Migration zur Verfügung.

## Aktivieren des Export-Modus

Die Dogu-Ressource besitzt in seiner Beschreibung ein boolesches Feld `exportMode`.
Um eine Dogu in den Export-Modus zu versetzen, muss dieses durch ein Update der Dogu-Ressource auf `true` gesetzt werden.

Beispiel:
```yaml
apiVersion: k8s.cloudogu.com/v2
kind: Dogu
metadata:
  name: ldap
  annotations:
    test: dev
  labels:
    dogu.name: ldap
    app: ces
spec:
  name: official/ldap
  version: 2.6.8-3
  exportMode: true
```

`kubectl apply -f ldap.yaml`

> **Hinweis:** Die Aktivierung des Export-Modus für zu einem Neustart des Dogus.

Technisch aktualisiert der `k8s-dogu-operator` das Deployment des Dogus. 
Es wird ein zusätzlicher "Sidecar"-Container zum Pod des Dogus hinzugefügt.
Dieser Container hat auch einen Volume-Mount für das Daten-Volume des Dogus.
Der stellt die Daten per "Rsync over SSH" für die Migration bereit

## Deaktivierung des Export-Modus

Um ein Dogu wieder in den Ausgangszustand zu versetzen, muss die Dogu-Ressource mit einem auf `false` gesetzen
`exportMode`-Feld aktualisiert werden.

Beispiel:
```yaml
apiVersion: k8s.cloudogu.com/v2
kind: Dogu
metadata:
  name: ldap
  annotations:
    test: dev
  labels:
    dogu.name: ldap
    app: ces
spec:
  name: official/ldap
  version: 2.6.8-3
  exportMode: false
```

`kubectl apply -f ldap.yaml`

Der zusätzliche Sidecar-Container wird jetzt wieder entfernt.

> **Hinweis:** Die Deaktivierung des Export-Modus für zu einem Neustart des Dogus.

