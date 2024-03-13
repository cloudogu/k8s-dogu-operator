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

> The sizes of the volumes must be specified in binary format (e.g. Mi or Gi).

Setting `dataVolumeSize` and updating the Dogu resource will start the process to increase the volume size.

Note that the value of `dataVolumeSize` must match the norm of 
[Kubernetes Quantities](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/).

In Kubernetes, however, a true increase in volume size is only possible if all pods using the volume, are shut down. As
a first step, the `k8s-dogu-operator` scales the deployment of the Dogus to **0** and shuts down **all** pods of the
Dogus shut down. Then the process to scale up the volume starts. The `k8s-dogu-operator` updates the desired size in
the `persistentVolumeClaim` of the Dogus. Then it waits until the storage controller enlarges the volume and the desired
size is reached. After that, the `k8s-dogu-operator` scales the deployment of the Dogus back to the original number of
replicas.

### Info
- Enlarging volumes can take several minutes to hours.
- Volumes cannot be scaled down.
