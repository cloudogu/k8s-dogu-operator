# Dogu-Volumes

In der Regel wird bei der Installation eines Dogus ein Volume mit einer Standardgröße erzeugt.
Die Größe des Volumes beträgt zwei Gigabyte. Über das Feld `minDataVolumeSize` kann initial eine definierte Größe
für ein Volume angegeben werden. Bei manchen Dogus kann es im späteren Betrieb allerdings sinnvoll sein
die Größe der Volumes zu bearbeiten.

## Vergrößern von Volumes

Die Dogu CR bietet dazu eine Konfigurationsmöglichkeit im Attribute `spec`:

Beispiel:

```yaml
spec:
  resources:
    minDataVolumeSize: 2Gi
```

> Die Größen der Volumes müssen im binären Format angegeben werden (z.B. Mi oder Gi).

Setzt man `minDataVolumeSize` und aktualisiert die Dogu-Ressource wird der Prozess zum Vergrößern des Volumes gestartet.

Zu beachten ist, dass der Wert von `minDataVolumeSize` der Norm von 
[Kubernetes-Quantities](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/) entsprechen 
muss.

In Kubernetes ist eine echte Vergrößerung des Volumes allerdings nur möglich, wenn alle Pods, die das Volume verwenden,
heruntergefahren werden. Als erster Schritt wird der `k8s-dogu-operator` das Deployment des Dogus auf **0** skaliert und
**alle** Pods des Dogus heruntergefahren. Anschließend startet der Prozess zur Vergrößerung des Volumes.
Der `k8s-dogu-operator` aktualisiert die gewünschte Größe im `persistentVolumeClaim` des Dogus. Anschließend wird
gewartet bis der Storage-Controller das Volume vergrößert und die gewünschte Größe erreicht wurde. Danach skaliert
der `k8s-dogu-operator` das Deployment des Dogus wieder auf die ursprüngliche Anzahl von Replicas.

### Info
- Das Vergrößern der Volumes kann mehrere Minuten bis Stunden in Anspruch nehmen.
- Volumes können nicht verkleinert werden.

## Aktuelle Größe als Status der Dogu-CR

Stellt der Controller fest, dass die Größe des Volumes verändert werden soll, so sind zu Beginn der Vergrößerung die konfigurierte `minDataVolumeSize` und die tatsächliche Größe des Volumes nicht identisch. Da Volumes nicht verkleinert werden dürfen, ist die `minDataVolumeSize` somit größer als die aktuelle Größe.

Dieser Zustand wird in der Condition `meetsMinVolumeSize` hinterlegt, gemeinsam mit dem Statusfeld `dogu.Status.DataVolumeSize`.
Vor dem Start hat die Condition den Wert `False`.

Im Zuge der eigentlichen Volume-Vergrößerung wird das Deployment zunächst auf 0 skaliert und hinterher wieder auf die konfigurierte Größe hochskaliert.
Dies dient dem Pod-Restart, so dass die PVCs aktualisiert eingebunden werden können. Dies kann einige Zeit dauern. 
Nachdem Neustart wird der Status erneut aktualisiert. Dabei wird solange gewartet, bis die tatsächliche Größe dem konfigurierten Minimum entspricht (oder größer).

Dies aktualisiert sowohl die Condition `meetsMinVolumeSize` auf `True` als auch den Wert des Statusfelds `dogu.Status.DataVolumeSize` auf die neue tatsächlich Größe.

## Vergrößern von Volumes von v3-Dogus

v3-Dogus konfigurieren ihre Volume-Größen in den `values` der Dogu-CR. Ist eine gerenderte Größe größer als die des
bestehenden PVCs, vergrößert der `k8s-dogu-operator` das PVC. Volumes können weder verkleinert noch in eine andere
StorageClass verschoben werden. Während ein Volume wächst, meldet das Dogu `Ready=False` mit dem Grund `ResizingPVC`.
`status.volumeStatus[].size` zeigt die neue Größe erst, wenn das Volume sie erreicht hat.

Der Operator vergrößert Volumes in einem von zwei Modi, den der Helm-Value `controllerManager.env.offlineVolumeExpansion`
(Env-Var `OFFLINE_VOLUME_EXPANSION`) für alle v3-Dogus festlegt.

### Online (Standard, `offlineVolumeExpansion: false`)

Der Operator patcht die PVCs, das Dogu läuft weiter. CSI-Treiber und Kubelet vergrößern Volume und Dateisystem der
laufenden Pods. StatefulSets mit veralteten `volumeClaimTemplates` werden mit Orphan-Propagation gelöscht und vom
Helm-Release neu angelegt; ihre Pods laufen weiter und werden vom neuen StatefulSet übernommen.

### Offline (`offlineVolumeExpansion: true`)

Für Speicher, der Volumes nur ungemountet vergrößern kann. Der Operator suspendiert das Helm-Release, patcht die PVCs,
löscht alle Workloads und Pods, die ein wachsendes PVC mounten, und setzt das Release anschließend fort. Das Dogu ist
nicht erreichbar, bis seine Workloads neu angelegt sind.

Bekannte Fälle, die Offline-Vergrößerung brauchen:
- vSphere CSI < 2.2
- OpenStack Cinder ohne Online-Extend
- Azure Disk Standard/Premium größer als 4 TiB
- Longhorn V2 Data Engine mit UBLK

Kubernetes erkennt diese Fälle nicht selbst. Im Online-Modus bleibt das Volume im Status `Resizing` bzw. `ResizeFailed` 
und das Dogu läuft mit der alten Größe weiter; Daten gehen nicht verloren.

### Im Online-Modus hängende Vergrößerung reparieren

Ein nachträglicher Wechsel in den Offline-Modus repariert eine online gestartete Vergrößerung nicht, weil das Release
bereits fortgesetzt ist. Einmalig von Hand reparieren:

1. `controllerManager.env.offlineVolumeExpansion: true` setzen und den Operator upgraden.
2. `spec.stopped: true` an der Dogu-CR setzen.
3. Warten, bis das PVC die Condition `FileSystemResizePending` oder `status.allocatedResourceStatuses[storage]=NodeResizePending`
   zeigt (`kubectl get pvc <name> -o yaml`).
4. `spec.stopped: false` setzen. Das Helm-Release legt die Workloads neu an, das Dateisystem wächst beim Mount durch die neuen Pods.