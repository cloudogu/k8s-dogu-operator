# Stopping, starting and restarting v3 dogus

v3 dogus are stopped and started via the dogu resource.
For a restart, there is a separate resource: `DoguRestart`.
In addition, the processing of a dogu can be paused, e.g. for maintenance.

The examples assume the namespace `ecosystem` and use the Nexus dogu.

## Stopping and starting

```bash
# stop
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge -p '{"spec":{"stopped":true}}'

# start
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge -p '{"spec":{"stopped":false}}'
```

### Stopping

With `spec.stopped: true`, all deployments and StatefulSets of the dogu are scaled to zero.
DaemonSets are not stopped.
The volumes of the dogu are retained, so no data is lost.

The stop is complete when the condition `Stopped` is `True`.
While stopping, it is `False` with the reason `PartiallyStopped`.
A stopped dogu is `Ready=True`, because the desired state is reached, and `Healthy=False` with the reason `Stopped`.

### Changes to a stopped dogu

A stopped dogu can still be configured or upgraded.
The changes are accepted and checked but only rolled out on the next start.
As long as changes are pending, the dogu shows `UpdatePending=True`.

### Starting

With `spec.stopped: false`, the dogu is started again.
All changes made during the stop are rolled out as well.
The start is complete when the dogu is `Ready=True` and `Healthy=True` again.

## Pausing the reconciliation

```bash
# pause
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge -p '{"spec":{"pauseReconciliation":true}}'

# resume
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge -p '{"spec":{"pauseReconciliation":false}}'
```

As long as `spec.pauseReconciliation` is set, changes to the dogu resource are neither checked nor rolled out.
This is useful, e.g., to collect several changes and roll them out together.
The running dogu stays unchanged.

- The dogu is `Ready=False` with the reason `ReconciliationPaused`.
- Pending changes are shown by `UpdatePending=True`.
- The pause takes precedence over stopping and starting: if `spec.stopped` is changed during the pause, it only takes effect after the pause ends.

With `spec.pauseReconciliation: false`, all pending changes are rolled out.

## Restarting

A v3 dogu is restarted via a `DoguRestart` resource.
All pods of the dogu are restarted; configuration and version stay unchanged.

```bash
kubectl -n ecosystem create -f - <<EOF
apiVersion: k8s.cloudogu.com/v2
kind: DoguRestart
metadata:
  generateName: nexus-restart-
spec:
  doguName: nexus
EOF
```

`generateName` creates a new resource with a unique name on every call.
The progress is shown by the conditions of the `DoguRestart` resource:

```bash
kubectl -n ecosystem get dogurestarts
kubectl -n ecosystem get dogurestart <name> \
  -o jsonpath='{range .status.conditions[*]}{.type}={.status} ({.reason}): {.message}{"\n"}{end}'
```

| Condition | Status  | Reason              | Meaning                                                                  |
|-----------|---------|---------------------|--------------------------------------------------------------------------|
| `Ready`   | `True`  | `Succeeded`         | All pods have been restarted.                                            |
| `Ready`   | `False` | `RolloutInProgress` | The pods are being restarted.                                            |
| `Ready`   | `False` | `DoguNotIdle`       | The dogu is being installed or upgraded; the restart waits.              |
| `Ready`   | `True`  | `DoguStopped`       | The dogu is stopped; the restart takes effect on the next start.         |
| `Ready`   | `True`  | `Superseded`        | A newer restart of the same dogu has already been applied.               |
| `Ready`   | `False` | `DoguNotFound`      | The dogu does not exist.                                                 |

Whether configuration changes trigger a restart by themselves is described in [Configuring v3 dogus](configuring_v3_dogus_en.md#restart-on-configuration-changes).
