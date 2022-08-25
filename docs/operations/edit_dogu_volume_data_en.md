# Adding or editing data in Dogu volumes.

## Thoughts on how to proceed

To edit data in a Dogu volume, the `kubectl cp` command can be used. This specifies the name of the pod and copies data
into it. So that one does not want to be dependent on a running Dogu container, it makes sense to start an extra Pod,
which takes over the copying and/or changing of the data. For the access of the Dogu data the Dogu volume is mounted at
this pod. This procedure makes it possible, for example, to edit data in a dogu even if it is in a faulty state. It is
also possible to mount data before a Dogu installation.

## Editing of Dogu volumes

From the consensus, there are two following use cases where Dogu volumes are edited.

### Editing data of an already installed dogu

For an installed dogu, its dogu volume already exists.
In this case, a matching pod must be created for the dogu in the cluster that mounts the dogu volume.

Example Redmine:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: dogu-redmine-volume-explorer
spec:
  containers:
    - image: alpine:3.16.2
      name: alpine-container
      command: [ 'sh', '-c', 'echo "Starting volume explorer!" && while sleep 3600; do :; done' ]
      volumeMounts:
        - mountPath: /volumes
          name: redmine-volume
  volumes:
    - name: redmine-volume
      persistentVolumeClaim:
        claimName: redmine
```

Pod creation:

```bash
kubectl apply -f <filename>.yaml
```

This pod mounts the Redmine volume under `/volumes`. Note that for other dogus, their volume names are the same as the
dogu name.

Once the pod is started you can now add data to the volume using `kubectl cp`.

Example Redmine plugin:

```bash
kubectl -n ecosystem cp redmine_dark/ dogu-redmine-volume-explorer:/volumes/plugins/
```

The behavior of dogu determines if it needs to be restarted. Then, the created pod can be removed from
the cluster again:

```bash
kubectl -n ecosystem delete pod dogu-redmine-volume-explorer
```

### Initial provisioning of data of a not yet installed Dogus

To initially provision data to dogus, the dogu volume itself must be created.

Example Redmine:

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

Volume creation:

```bash
kubectl apply -f <filename>.yaml
```

The provisioner, labels and storage class are validated by the `dogu-operator` and must not be changed.
The size of the volume can be adjusted as desired.

After creating the volume, copy data to the volume using the above procedure. After that the Dogu can be installed.
The `dogu-operator` recognizes during the installation that a volume already exists for the Dogu and uses it.
