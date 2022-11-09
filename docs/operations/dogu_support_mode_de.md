# Support-Strategie für Dogus

Unter bestimmten Bedingungen kann es vorkommen, dass die Pods der Dogus sich in einer Neustart-Schleife befinden.
In solchen Fällen ist es hilfreich sich per Shell mit dem Container zu verbinden und das Filesystem zu analysieren.
Der Support-Modus unterbindet die Neustart-Schleife und versetzt das Dogu in einen "eingefrorenen" Modus, damit die
Verbindung zu dem Container ermöglicht wird.

## Aktivieren des Support-Modus

Die Dogu-Ressource besitzt in seiner Beschreibung ein boolesches Feld `supportMode`.
Um eine Dogu in den Support-Modus zu versetzen, muss dieses durch ein Update der Dogu-Ressource auf `true` gesetzt werden.

Beispiel:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: postfix
  annotations:
    test: dev
  labels:
    dogu.name: postfix
    app: ces
spec:
  name: official/postfix
  version: 3.6.4-3
  supportMode: true
```

`kubectl apply -f postfix.yaml`

Hierbei werden andere Änderungen der Dogu-Beschreibung ignoriert. Ebenfalls, wenn sich ein Dogu bereits in dem Support-Modus
befindet.

Technisch aktualisiert der `k8s-dogu-operator` das Deployment des Dogus. Der Startup-Befehl des Containers wird
ignoriert, indem ein Sleep-Command hinzugefügt wird. Die gewöhnlichen Probes des Containers werden gelöscht, damit in der
Wartung nicht der Container von dem Pod-Controller neu gestartet wird. Außerdem wird in dem Container eine Umgebungsvariable
`SUPPORT_MODE` auf `true` gesetzt. Nach der Aktualisierung des Deployments werden die Pods des Dogus neu gestartet und man 
kann sich mit ihnen verbinden.

Beispiel:

`k exec -it postfix-<pod_id> -- sh`

## Deaktivierung des Support-Modus

Um ein Dogu wieder in den Ausgangszustand zu versetzen, muss die Dogu-Ressource mit einem auf `false` gesetzen
`supportMode`-Feld aktualisiert werden.

Beispiel:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: postfix
  annotations:
    test: dev
  labels:
    dogu.name: postfix
    app: ces
spec:
  name: official/postfix
  version: 3.6.4-3
  supportMode: false
```

`kubectl apply -f postfix.yaml`
