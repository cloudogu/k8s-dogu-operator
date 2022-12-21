# Dogu-Volumes

In der Regel wird bei der Installation eines Dogus ein Volume mit einer Standardgröße erzeugt.
Die Größe des Volumes beträgt zwei Gigabyte. Bei manchen Dogus kann es im späteren Betrieb allerdings sinnvoll sein
die Größe der Volumes zu bearbeiten.

## Vergrößern von Volumes

Die Dogu CR bietet dazu eine Konfigurationsmöglichkeit im Attribute `spec`:

Beispiel:

```yaml
spec:
  resources:
    dataVolumeSize: 2Gi
```

Setzt man `dataVolumeSize` und aktualisiert die Dogu-Ressource wird der Prozess zum Vergrößern des Volumes gestartet.

Zu beachten ist, dass der Wert von `dataVolumeSize` der Norm von 
[Kubernetes-Quantities](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/) entsprechen 
muss.

Startet der Prozess zur Vergrößerung des Volumes, wird zunächst der `dogu-operator` das `persistentVolumeClaim` des
Dogus selektieren und die neue Größe aktualisieren. In Kubernetes is eine echte Vergrößerung des Volumes allerdings nur
möglich, wenn alle Pods, die das Volume verwenden, heruntergefahren werden. Als nächsten Schritt wird der `dogu-operator`
das Deployment des Dogus auf **0** skalieren und **alle** Dogu-Pods herunterfahren. Anschließend wird gewartet bis der 
Storage-Controller das Volume vergrößert und danach wieder auf die ursprüngliche Anzahl von Replicas hochskaliert.

### Info
- Das Vergrößern der Volumes kann mehrere Minuten bis Stunden in Anspruch nehmen.
- Volumes können nicht verkleinert werden.