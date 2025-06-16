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

Stellt der Kontroller fest, dass die Größe des Volumes verändert werden soll, so sind zu Beginn der Vergrößerung die konfigurierte
`minDataVolumeSize` und die tatsächliche Größe des Volumes nicht identisch. Da Volumes nicht verkleinert werden dürfen, ist die `minDataVolumeSize` 
somit größer als die aktuelle Größe.

Dieser Zustand wird in der Condition `MeetsMinimumDataVolumeSize` hinterlegt, gemeinsam mit dem Statusfeld `dogu.Status.DataVolumeSize`.
Vor dem Start hat die Condition den Wert `False`.

Im Zuge der eigentlichen Volume-Vergrößerung wird das Deployments zunächst auf 0 skaliert und hinterher wieder auf die konfigurierte Größe hochskaliert.
Dies dient dem Pod-Restart, so dass die PVCs aktualiert eingebunden werden können. Dies kann einige Zeit dauern. 
Nachdem Neustart wird der Status erneut aktualiert. Dabei wird solange gewartet, bis die tatschliche Größe dem konfigurierten Minimum entsprecht (oder größer).

Dies aktualisert sowohl die Condition `MeetsMinimumDataVolumeSize` auf `True` als auch den Wert des Statusfelds `dogu.Status.DataVolumeSize` 
auf die neue tatsächlich Größe.