# v3-Dogus stoppen, starten und neu starten

v3-Dogus werden über die Dogu-Ressource gestoppt und gestartet.
Für einen Neustart gibt es eine eigene Ressource: `DoguRestart`.
Zusätzlich lässt sich die Verarbeitung eines Dogus pausieren, z. B. für Wartungsarbeiten.

Die Beispiele gehen vom Namespace `ecosystem` aus und verwenden das Nexus-Dogu.

## Stoppen und starten

```bash
# stoppen
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge -p '{"spec":{"stopped":true}}'

# starten
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge -p '{"spec":{"stopped":false}}'
```

### Stoppen

Mit `spec.stopped: true` werden alle Deployments und StatefulSets des Dogus auf null skaliert.
DaemonSets werden nicht gestoppt.
Die Volumes des Dogus bleiben erhalten, es gehen also keine Daten verloren.

Der Stopp ist abgeschlossen, wenn die Condition `Stopped` auf `True` steht.
Während des Stoppens ist sie `False` mit dem Reason `PartiallyStopped`.
Ein gestopptes Dogu ist `Ready=True`, weil der gewünschte Zustand erreicht ist, und `Healthy=False` mit dem Reason `Stopped`.

### Änderungen an einem gestoppten Dogu

Ein gestopptes Dogu kann weiterhin konfiguriert oder aktualisiert werden.
Die Änderungen werden angenommen und geprüft, aber erst beim nächsten Start ausgerollt.
Solange Änderungen ausstehen, zeigt das Dogu `UpdatePending=True`.

### Starten

Mit `spec.stopped: false` wird das Dogu wieder gestartet.
Dabei werden auch alle Änderungen ausgerollt, die während des Stopps angefallen sind.
Der Start ist abgeschlossen, wenn das Dogu wieder `Ready=True` und `Healthy=True` ist.

## Reconciliation pausieren

```bash
# pausieren
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge -p '{"spec":{"pauseReconciliation":true}}'

# fortsetzen
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge -p '{"spec":{"pauseReconciliation":false}}'
```

Solange `spec.pauseReconciliation` gesetzt ist, werden Änderungen an der Dogu-Ressource weder geprüft noch ausgerollt.
Das ist z. B. hilfreich, um mehrere Änderungen zu sammeln und gemeinsam auszurollen.
Das laufende Dogu bleibt dabei unverändert.

- Das Dogu ist `Ready=False` mit dem Reason `ReconciliationPaused`.
- Ausstehende Änderungen zeigt `UpdatePending=True`.
- Die Pause hat Vorrang vor Stoppen und Starten: Wird `spec.stopped` während der Pause geändert, wirkt das erst nach dem Ende der Pause.

Mit `spec.pauseReconciliation: false` werden alle ausstehenden Änderungen ausgerollt.

## Neu starten

Ein v3-Dogu wird über eine `DoguRestart`-Ressource neu gestartet.
Dabei werden alle Pods des Dogus neu gestartet; Konfiguration und Version bleiben unverändert.

```bash
kubectl -n ecosystem create -f - <<EOF
apiVersion: k8s.cloudogu.com/v2
kind: DoguRestart
metadata:
  generateName: nexus-restart-
spec:
  doguName: nexus
EOF
```

`generateName` erzeugt bei jedem Aufruf eine neue Ressource mit eindeutigem Namen.
Den Fortschritt zeigen die Conditions der `DoguRestart`-Ressource:

```bash
kubectl -n ecosystem get dogurestarts
kubectl -n ecosystem get dogurestart <name> \
  -o jsonpath='{range .status.conditions[*]}{.type}={.status} ({.reason}): {.message}{"\n"}{end}'
```

| Condition   | Status  | Reason              | Bedeutung                                                                |
|-------------|---------|---------------------|--------------------------------------------------------------------------|
| `Ready`     | `True`  | `Succeeded`         | Alle Pods sind neu gestartet.                                            |
| `Ready`     | `False` | `RolloutInProgress` | Die Pods werden neu gestartet.                                           |
| `Ready`     | `False` | `DoguNotIdle`       | Das Dogu wird gerade installiert oder aktualisiert; der Neustart wartet. |
| `Ready`     | `True`  | `DoguStopped`       | Das Dogu ist gestoppt; der Neustart wirkt beim nächsten Start.           |
| `Ready`     | `True`  | `Superseded`        | Ein neuerer Neustart desselben Dogus wurde bereits ausgeführt.           |
| `Ready`     | `False` | `DoguNotFound`      | Das Dogu existiert nicht.                                                |

Ob Konfigurationsänderungen von selbst einen Neustart auslösen, steht in [v3-Dogus konfigurieren](configuring_v3_dogus_de.md#neustart-bei-konfigurationsänderungen).
