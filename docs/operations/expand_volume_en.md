# Dogu volumes

Usually, a volume with a default size is created during the installation of a Dogu.
The size of the volume is two gigabytes. The `dataVolumeSize` field can be used to specify a custom size for a volume.
However, for some dogus it may be useful to edit the volume size later on.

## Increasing the size of volumes

The Dogu CR provides a configuration option for this in the `spec` attribute:

Example:

```yaml
spec:
  resources:
    dataVolumeSize: 2Gi
```

Setting `dataVolumeSize` and updating the Dogu resource will start the process to increase the volume size.

Note that the value of `dataVolumeSize` must match the norm of 
[Kubernetes Quantities](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/).

If the process starts to increase the size of the volume, first the `dogu-operator` will select the `persistentVolumeClaim` 
of the Dogus and update the new size. In Kubernetes, however, true volume growth is only possible if all the pods using
the volume are shut down. The next step the `dogu-operator` will scale the deployment of the dogu to **0** and shut 
down **all** dogu pods. Then it will wait until the storage controller increases the volume and then scales it back up 
to the original number of replicas.

### Info
- Enlarging volumes can take several minutes to hours.
- Volumes cannot be scaled down.
