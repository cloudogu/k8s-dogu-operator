# Configuration for Installation of v3 Dogus

## Flux Dependencies

v3 dogus are installed as Helm charts with the help of two Flux controllers: source-controller and helm-controller.
Both have to be installed, together with their CRDs in order to allow v3 dogus to be installed. This can be done
using ecosystem-core by setting its `flux.enabled` value to `true`. Besides install and upgrade operations this also 
enables drift detection for dogu installations, which automatically reverts any manual changes to related cluster state.

## Configuration

The dogu operator needs some additional configuration for locating and retrieving the artifacts for v3 dogus:

The secret `dogu-registry-v3` has to contain the URL and credentials for the v3 dogu registry 
(see [Configuring the Dogu V3 Registry](configuring_the_dogu_registry_en.md)). 

In addition there are some configuration options to configure how the Helm installation is handled: 
 - `DOGU_HELM_RETRY_INTERVAL` controls how long to wait until attempting a retry, if the Helm install or upgrade fails. 
 - `DOGU_HELM_RECONCILIATION_INTERVAL` defines the interval in which to check for any drift from the state requested
    by Helm, which is then automatically corrected.

Both intervals are given as a number with a time unit, e.g. "20s" (for 20 seconds), "2m" (for 2 minutes), 
or "1h" (for 1 hour). Combinations like "1h20m" (for 1 hour and 20 minutes) are also possible.

## Stopping and Starting

- **Stop:** `spec.stopped: true` scales Deployments and StatefulSets to zero and retains StatefulSet PVCs during scale-down.
  `Stopped=True` confirms that the associated Pods are gone. DaemonSets are not supported.
- **Changes:** The OCIRepository and HelmRelease are updated while stopped, but changes are not rolled out yet.
- **Start:** `spec.stopped: false` prepares the current desired state. After suspension is lifted, Flux rolls it out;
  a successful release and a current `Healthy=True` confirm the start.
- **Pause:** `spec.pauseReconciliation: true` pauses resource changes by the dogu operator, but pending checks continue.
  The HelmRelease must also be suspended to prevent corrections by Flux.
- **UpdatePending:** `True` means unapplied changes, `False` no differences, and `Unknown` an incomplete comparison.
  The chart source check is not implemented yet, so the status remains `Unknown` unless a difference is detected.

Stopping requires Flux to acknowledge `spec.suspend: true` on the HelmRelease; starting requires it to be set to `false` separately.
For initially stopped dogus, the HelmRelease is created suspended. Global configuration changes are considered during the next reconciliation.
