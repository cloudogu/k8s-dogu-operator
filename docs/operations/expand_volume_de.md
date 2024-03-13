# Dogu-Volumes

In der Regel wird bei der Installation eines Dogus ein Volume mit einer Standardgröße erzeugt.
Die Größe des Volumes beträgt zwei Gigabyte. Über das Feld `dataVolumeSize` kann initial eine definierte Größe
für ein Volume angegeben werden. Bei manchen Dogus kann es im späteren Betrieb allerdings sinnvoll sein
die Größe der Volumes zu bearbeiten.

## Vergrößern von Volumes

Die Dogu CR bietet dazu eine Konfigurationsmöglichkeit im Attribute `spec`:

Beispiel:

```yaml
spec:
  resources:
    dataVolumeSize: 2Gi
```

> Die Größen der Volumes müssen im binären Format angegeben werden (z.B. Mi oder Gi).

Setzt man `dataVolumeSize` und aktualisiert die Dogu-Ressource wird der Prozess zum Vergrößern des Volumes gestartet.

Zu beachten ist, dass der Wert von `dataVolumeSize` der Norm von 
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