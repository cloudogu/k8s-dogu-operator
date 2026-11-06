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

Set `spec.stopped: true` on the Dogu resource to stop its workloads. The operator waits for HelmRelease suspension,
scales Deployments and StatefulSets to zero and waits for their pods to disappear. StatefulSet PVCs are retained.
`status.stopped: true` confirms that the workloads have stopped.

While stopped, the operator still updates chart sources and HelmRelease inputs; Flux does not apply them while suspended.
Set `spec.stopped: false` to start the dogu again. The stopped status is cleared after deployment and health checks succeed.

`spec.pauseReconciliation: true` takes precedence over starting or stopping: regular resource changes are skipped,
but source and release checks can still set `UpdatePending=True`. A current-generation True is retained while paused
or stopped. An older condition is cleared only after all relevant checks complete without detecting pending changes.
An incomplete check leaves the condition unchanged and retries.
