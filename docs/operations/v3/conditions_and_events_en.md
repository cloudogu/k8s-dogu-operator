# Conditions and events of v3 dogus

The state of a v3 dogu is shown in the conditions of the dogu resource.
Each condition describes one aspect, e.g. whether the installation is complete (`Ready`) or whether the pods are running (`Healthy`).
The status is `True` or `False`, the reason names the cause and the message provides details.
On success, the reason is always `Succeeded`.

In addition, the dogu operator writes events to the dogu resource, e.g. when an operation completes or fails.

```bash
# Conditions
kubectl -n ecosystem get dogus.v3beta1.k8s.cloudogu.com <dogu> \
  -o jsonpath='{range .status.conditions[*]}{.type}={.status} ({.reason}): {.message}{"\n"}{end}'

# Events
kubectl -n ecosystem get events --field-selector involvedObject.kind=Dogu,involvedObject.name=<dogu>
```

## Conditions

### `Ready`

`Ready` describes the overall state of the dogu resource.
It is `True` when the desired state of the dogu resource is fully applied.
With `False`, the reason names the operation in progress or the error that occurred:

| Reason                                                                   | Meaning                                                                         |
|--------------------------------------------------------------------------|---------------------------------------------------------------------------------|
| `Installing`, `Upgrading`, `Starting`, `Stopping`, `ResizingPVC`         | An operation is in progress; no action needed.                                  |
| `WorkloadsNotReady`                                                      | The dogu is rolled out, but not all pods are ready yet.                         |
| `ReconciliationPaused`                                                   | `spec.pauseReconciliation` is set.                                              |
| `DownloadFailed`                                                         | The dogu could not be fetched from the dogu registry.                           |
| `Unauthorized`                                                           | No access to the chart registry (`ces-container-registries`).                   |
| `ChartNotFound`                                                          | The version does not exist as a v3 dogu.                                        |
| `SchemaInvalid`, `RenderFailed`, `VolumeShrink`, `StorageClassImmutable` | Invalid configuration, see [Validation](configuring_v3_dogus_en.md#validation). |
| `InstallFailed`, `UpgradeFailed`                                         | Installation or upgrade failed; retried automatically.                          |
| `MigrationPending`, `MigrationFailed`                                    | See [Migrating from v2](migrating_from_v2_to_v3_en.md).                         |
| `Deleting`, `DeletionFailed`, `DeletionStalled`                          | The dogu is being deleted or the deletion failed or is blocked.                 |
| `DowngradeNotAllowed`                                                    | A migration back from v3 to v2 is not supported.                                |
| `StepFailed`, `ReconciliationAborted`                                    | Generic error; message and operator log name the cause.                         |

If an operation fails, the dogu usually keeps running with its previous state.
Once the cause is fixed, the operator continues the operation automatically.

### Further conditions

| Condition                 | `True` means                                             | Reasons for `False` or details                                                   |
|---------------------------|----------------------------------------------------------|----------------------------------------------------------------------------------|
| `Healthy`                 | All pods of the dogu are ready.                          | `WorkloadsNotReady`, `Stopped`                                                   |
| `ChartAvailable`          | The chart of the dogu has been pulled.                   | `DownloadFailed`, `Unauthorized`, `ChartNotFound`                                |
| `Valid`                   | The configuration is valid.                              | `SchemaInvalid`, `RenderFailed`, `VolumeShrink`, `StorageClassImmutable`         |
| `SchemaValidationSkipped` | `spec.skipSchemaValidation` is set.                      | `SchemaValidationEnabled`                                                        |
| `Stopped`                 | The dogu is fully stopped.                               | `PartiallyStopped` (stopping in progress), `NotStopped`                          |
| `Suspended`               | Changes are not rolled out.                              | Reasons for `True`: `DoguStopped`, `ReconciliationPaused`, `PVCResizeInProgress` |
| `UpdatePending`           | There are changes that take effect after start or pause. | Reasons: `Stopped`, `ReconciliationPaused`                                       |
| `ExportModeActive`        | –                                                        | Not implemented for v3 dogus yet (`NotImplemented`)                              |

`Healthy` is checked independently of `Ready`.
A dogu can therefore be `Ready=True` and `Healthy=False` at the same time, e.g. when it is stopped or a pod crashes.

## Events

Events record operations and state changes of the dogu and are mainly used for retrospective troubleshooting.

| Type    | Reason                         | Trigger                                               |
|---------|--------------------------------|-------------------------------------------------------|
| Normal  | `ReconcileSucceeded`           | The desired state is applied.                         |
| Normal  | `ChartAvailable`               | The chart was pulled or the installation was updated. |
| Warning | Reason of `Ready`              | An operation was aborted, e.g. `SchemaInvalid`.       |
| Normal  | `Succeeded`                    | `Healthy` changes to `True`.                          |
| Warning | `WorkloadsNotReady`, `Stopped` | `Healthy` changes to `False`.                         |
| Normal  | `UpdatePending`                | A change waits for the end of the pause.              |
| Normal  | `ResizingPVC`                  | A volume is being expanded.                           |

Automatic retries and pauses produce no warning events.
The current state is always in the conditions.
Events about restarts are written to the respective `DoguRestart` resource, not to the dogu.
