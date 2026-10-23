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

## Increasing the size of volumes of v3 dogus

v3 dogus configure their volume sizes in the `values` of the Dogu CR. If a rendered size is larger than the size of the
existing PVC, the `k8s-dogu-operator` grows the PVC. Volumes can neither be shrunk nor moved to another StorageClass.
While a volume grows, the dogu reports `Ready=False` with the reason `ResizingPVC`. `status.volumeStatus[].size` only
shows the new size once the volume has reached it.

The operator grows volumes in one of two modes, which the Helm value `controllerManager.env.offlineVolumeExpansion`
(env var `OFFLINE_VOLUME_EXPANSION`) selects for all v3 dogus.

### Online (default, `offlineVolumeExpansion: false`)

The operator patches the PVCs and the dogu keeps running. The CSI driver and the kubelet grow the volume and the
file system of the running pods. StatefulSets whose `volumeClaimTemplates` are outdated are deleted with orphan
propagation and recreated by the Helm release; their pods keep running and are adopted by the new StatefulSet.

### Offline (`offlineVolumeExpansion: true`)

For storage that can only grow unmounted volumes. The operator suspends the Helm release, patches the PVCs, deletes
every workload and pod that mounts a growing PVC and resumes the release afterwards. The dogu is unavailable until its
workloads are recreated.

Known cases that need offline expansion:
- vSphere CSI < 2.2
- OpenStack Cinder without online extend
- Azure Disk Standard/Premium larger than 4 TiB
- Longhorn V2 data engine with UBLK

Kubernetes cannot detect these cases itself. In online mode the volume stays in state `Resizing` or `ResizeFailed` and 
the dogu keeps running with the old size; no data is lost.

### Repairing a resize that got stuck in online mode

Switching to offline mode afterwards does not repair a resize that was started online, because the release is already
resumed. Repair it once by hand:

1. Set `controllerManager.env.offlineVolumeExpansion: true` and upgrade the operator.
2. Set `spec.stopped: true` on the Dogu CR.
4. Wait until the PVC shows the condition `FileSystemResizePending` or `status.allocatedResourceStatuses[storage]=NodeResizePending`
   (`kubectl get pvc <name> -o yaml`).
5. Set `spec.stopped: false`. The Helm release recreates the workloads, and the file system grows when the new pods mount the volume.
