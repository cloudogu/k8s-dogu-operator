# Hinzufügen oder Editieren von Daten in Dogu-Volumes

## Gedanken zum Vorgehen

Um Daten in einem Dogu-Volume zu bearbeiten, kann der Befehl `kubectl cp` verwendet werden. Dabei wird die Referenz des 
Pods angegeben und in diesen Daten kopiert. Damit man nicht abhängig von einem `running` Dogu-Container sein möchte, ist
es sinnvoll einen extra Pod zu starten, der das Kopieren bzw. Verändern der Daten übernimmt. Für den Zugriff der 
Dogu-Daten wird an diesem Pod das Dogu-Volume gemounted. Dieses Vorgehen ermöglicht es z.B. Daten in einem Dogu zu 
bearbeiten, auch wenn es in einem fehlerhaften Zustand ist.

## Bearbeitung von Dogu-Volumes

Aus dem allgemeinen Konsens ergeben sich zwei folgende Anwendungsfälle bei denen Dogu-Volumes bearbeitet werden.

### Bearbeitung von Daten eines bereits installierten Dogus

Bei einem installierten Dogu existiert bereits sein Dogu-Volume.
Hierbei muss für das Dogu ein passender Pod im Cluster erstellt werden, der das Dogu-Volume einbindet.

Beispiel Redmine:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: dogu-redmine-volume-explorer
spec:
  containers:
    - image: alpine:3.16.2
      name: alpine-container
      command: ['sh', '-c', 'echo "Starting volume explorer!" && while sleep 3600; do :; done']
      volumeMounts:
        - mountPath: /volumes
          name: redmine-volume
  volumes:
    - name: redmine-volume
      persistentVolumeClaim:
        claimName: redmine
```

Erstellung des Pods:
`kubectl apply -f <filename>.yaml`

Dieser Pod bindet das Redmine-Volume unter `/volumes` ein. Zu beachten ist, dass für andere Dogus deren Volume-Namen der
Dogu-Namen entsprechen.

Ist der Pod gestartet kann man nun über `kubectl cp` Daten in das Volume hinzufügen.

Beispiel Redmine-Plugin:
`kubectl -n ecosystem cp redmine_dark/ dogu-redmine-volume-explorer:/volumes/plugins/`

Das Verhalten des Dogus bestimmt ob dieses anschließend neu gestartet werden muss.
Anschließend kann der erstellte Pod wieder aus dem Cluster entfernt werden:
`kubectl -n ecosystem delete pod dogu-redmine-volume-explorer`

### Initiale Bereitstellung von Daten eines noch nicht installierten Dogus

Um Daten initial für Dogus bereitzustellen, muss das Dogu-Volume selber erzeugt werden.

Beispiel Redmine:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  annotations:
    volume.beta.kubernetes.io/storage-provisioner: driver.longhorn.io
    volume.kubernetes.io/storage-provisioner: driver.longhorn.io
  labels:
    app: ces
    dogu: redmine
  name: redmine
  namespace: ecosystem
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
  storageClassName: longhorn
```

Erstellung des Volumes:
`kubectl apply -f <filename>.yaml`

Die Provisioner, Labels und die Storageclass werden von dem `dogu-operator` validiert und dürfen nicht verändert werden.
Die Größe des Volumes kann beliebig angepasst werden.

Nach der Erstellung des Volumes kopiert man mit dem obigen Vorgehen Daten in das Volume. Danach kann das Dogu 
installiert werden. Der `dogu-operator` erkennt bei der Installation, dass für das Dogu bereits ein Volume existiert und 
verwendet es.
