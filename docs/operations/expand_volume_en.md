# Dogu volumes

Usually, a volume with a default size is created during the installation of a Dogu.
The size of the volume is two gigabytes. The `minDataVolumeSize` field can be used to specify a custom size for a volume.
However, for some dogus it may be useful to edit the volume size later on.

## Increasing the size of volumes

The Dogu CR provides a configuration option for this in the `spec` attribute:

Example:

```yaml
spec:
  resources:
    minDataVolumeSize: 2Gi
```

> The sizes of the volumes must be specified in binary format (e.g. Mi or Gi).

Setting `minDataVolumeSize` and updating the Dogu resource will start the process to increase the volume size.

Note that the value of `minDataVolumeSize` must match the norm of 
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

## Current size as the status of the Dogu-CR

If the controller determines that the volume size needs to be changed, the configured
`minDataVolumeSize` and the actual size of the volume are not identical at the start of the resize process. 
Since volumes cannot be reduced in size, the `minDataVolumeSize` is therefore larger than the current size.

This state is stored in the condition `meetsMinVolumeSize`, together with the status field `dogu.Status.DataVolumeSize`.

Before starting, the condition has the value `False`.

During the actual volume resize, the deployment is first scaled to 0 and then scaled back up to the configured size.
This is used for the pod restart so that the PVCs can be updated and mounted. This can take some time.

After the restart, the status is updated again. The system waits until the actual size equals (or exceeds) the configured minimum.

This updates both the condition `meetsMinVolumeSize` to `True` and the value of the status field `dogu.Status.DataVolumeSize`
to the new actual size.
