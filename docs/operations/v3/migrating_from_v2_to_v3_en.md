# Migrating from v2

An installed v2 dogu is migrated to v3 by setting `spec.doguApiVersion: v3` on its dogu resource.
The dogu operator detects the migration on its own and carries it out completely.
The data of the dogu's volume and the credentials of its service accounts are taken over.
The configuration of the v2 dogu is not taken over.

> [!WARNING]
> There is no way back to v2.
> After a successful migration the operator deletes the volume of the v2 dogu.
> A backup should be created before the migration.

The examples assume the namespace `ecosystem` and use the Nexus dogu.

## Sequence

1. The operator checks whether the v2 dogu can be migrated, see [Prerequisites](#prerequisites).
2. It pulls the chart of the target version and validates it together with the values.
   If this fails, the v2 dogu remains untouched.
3. It copies the credentials of the service accounts from the sensitive config of the v2 dogu into the secret
   `<dogu>-migration`.
4. It scales the v2 dogu down to 0. **The downtime of the dogu starts here.**
   The volume of the v2 dogu is kept and bound to the new claim `<dogu>-migration`.
5. It deletes the remaining resources of the v2 dogu: config, sensitive config, deployment, services,
   network policies and the `AuthRegistration`, `Exposition` and `WarpMenuEntry` resources.
6. If the chart ships migration jobs, the operator installs the chart with the migration jobs enabled.
   The jobs take over the data from the claim `<dogu>-migration` and, if needed, from shared v2 dogus such as `postgresql`.
   The application of the dogu does not start during this time.
   The operator waits until all migration jobs have completed.
7. It removes the secret and the claim `<dogu>-migration`.
   The volume of the v2 dogu and its data are deleted, see [After the migration](#after-the-migration).
8. It installs the chart the regular way. The dogu is then an ordinary v3 dogu.

An interrupted migration, e.g. due to a restart of the operator, is resumed at the point where it stopped.

If the v2 dogu has a volume, the chart of the target version must ship migration jobs.
Otherwise, the migration fails with `MigrationFailed` before the data would be lost.

## Preparation

### Prerequisites

- v3 dogus are enabled in the operator, see [Installing v3 dogus](installing_v3_dogus_en.md#prerequisites).
- The target version exists as a v3 dogu and supports the migration from v2, see the release notes of the dogu.
- The v2 dogu is installed and no operation is running.
- Support mode and export mode are not active and the volume of the dogu is not being resized.

If one of the last two points is not met, the operator does not start the migration.
The dogu then stays `Ready=False` with the reason `MigrationPending` and the message `The migration to v3 is blocked by the v2 dogu: <reason>`.
The operator checks again every minute and starts the migration as soon as the cause is gone.

### Transferring the configuration

The operator does not take over the configuration of the v2 dogu.
The config and the sensitive config of the v2 dogu are deleted during the migration.
Only the credentials of the service accounts are kept.

Before the migration, look up the configuration of the v2 dogu and transfer it to `spec.values` or
`spec.mappedValues`, see [Configuring v3 dogus](configuring_v3_dogus_en.md):

```bash
kubectl -n ecosystem get configmap nexus-config -o jsonpath='{.data.config\.yaml}'
```

The values are set in the same change as `spec.doguApiVersion`, see [Starting the migration](#starting-the-migration).
This way the dogu starts with its configuration right after the migration.

### Backup

Back up the dogu before the migration, including its data in shared dogus such as `postgresql`.
If the migration cannot be completed, the backup is the only way back to a working state.

### Storage

During the migration, the volume of the v2 dogu exists next to the new volumes of the v3 dogu.
Plan for twice the storage capacity until the migration is complete.

The new volumes must be at least as large as the volume of the v2 dogu, so that the data fits.
Their size is set via the values of the dogu; the operator does not check it.

### Timeouts of the migration jobs

The operator waits only for a limited time for the migration jobs:

| Helm value (`controllerManager.env.`) | Default | Meaning                                                                    |
|---------------------------------------|---------|----------------------------------------------------------------------------|
| `migrationJobStartTimeout`            | `5m`    | Time from the creation of the HelmRelease until the first migration job appears. |
| `migrationJobCompletionTimeout`       | `1h`    | Time a single migration job may run.                                       |

For large data volumes, increase `migrationJobCompletionTimeout` before the migration.
The values are set via the component resource of the operator, like the [further operator settings](installing_v3_dogus_en.md#further-operator-settings).

## Starting the migration

The migration starts as soon as the dogu resource contains `spec.doguApiVersion: v3`.
Set the version of the v3 dogu and the configuration in the same change:

```bash
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge -p '
spec:
  doguApiVersion: v3
  version: <version>
  values:
    # configuration of the dogu
'
```

## Following the migration

During the migration the dogu is `Ready=False` with the reason `MigrationPending`.
The message of the condition shows the current step:

```bash
kubectl -n ecosystem get dogus.v3beta1.k8s.cloudogu.com nexus \
  -o jsonpath='{range .status.conditions[?(@.type=="Ready")]}{.reason}: {.message}{"\n"}{end}'
```

```text
MigrationPending: Waiting for 1 of 2 migration jobs to complete
```

The events show the course of the migration with timestamps:

```bash
kubectl -n ecosystem get events --field-selector involvedObject.kind=Dogu,involvedObject.name=nexus
```

| Type    | Reason             | Message (examples)                                                          |
|---------|--------------------|-----------------------------------------------------------------------------|
| Normal  | `MigrationPending` | `Migration secret "nexus-migration" created`                                |
| Normal  | `MigrationPending` | `Prepared the dogu pvc for the v3 migration`                                |
| Normal  | `MigrationPending` | `Initiated deletion of 7 resources of the v2 installation`                  |
| Normal  | `MigrationPending` | `Migration network policy "nexus-migration-postgresql" created`             |
| Normal  | `MigrationPending` | `Waiting for 1 of 2 migration jobs to complete`                             |
| Normal  | `MigrationPending` | `All 2 migration jobs completed`                                            |
| Normal  | `MigrationPending` | `Migration resources removed`                                               |
| Normal  | `ReconcileSucceeded` | `Dogu reconciliation completed successfully`                              |
| Warning | `MigrationPending` | A step failed temporarily; the operator retries it automatically.           |
| Warning | `MigrationFailed`  | The migration was aborted, see [Failed migration](#failed-migration).       |

The migration jobs and their logs:

```bash
kubectl -n ecosystem get jobs -l k8s.cloudogu.com/migration-job=true,k8s.cloudogu.com/dogu.name=nexus
kubectl -n ecosystem logs job/<job>
```

The migration is complete when the dogu is `Ready=True` with the reason `Succeeded`.
Then, as with an installation, the dogu starts and becomes `Healthy=True`.

## Failed migration

If the migration cannot be completed, the dogu stays `Ready=False` with the reason `MigrationFailed`.
Additionally, there is a warning event with the same message.
The operator does not retry the migration on its own.

The data is not lost: the secret and the claim `<dogu>-migration` and the volume of the v2 dogu are kept.

| Message                                                                         | Cause / solution                                                                                                                                                                                                    |
|---------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `Migration job "<job>" failed: ...`                                             | A migration job failed. Check the logs of the job, fix the cause and [restart the migration jobs](#restarting-the-migration-jobs).                                                                                     |
| `Migration job "<job>" did not complete within <duration>`                      | A migration job took longer than `migrationJobCompletionTimeout`. Increase the timeout and [restart the migration jobs](#restarting-the-migration-jobs).                                                               |
| `No migration job of dogu "<dogu>" found within <duration>`                     | No migration job appeared within `migrationJobStartTimeout`. Check the HelmRelease of the dogu (`kubectl -n ecosystem get helmrelease nexus`), fix the cause and [restart the migration jobs](#restarting-the-migration-jobs). |
| `the chart ships no migration job, so the data of the migration volume claim ...` | The target version does not support the migration. Set `spec.version` to a version that supports it; the migration then continues automatically.                                                                   |
| `failed to parse sensitive config: ...`, `... both result in the migration secret key ...` | The sensitive config of the v2 dogu (`<dogu>-config`) cannot be read. Correct the secret; the migration then continues automatically. The v2 dogu is still running in this case.                                 |

### Restarting the migration jobs

The migration jobs are run again by deleting the HelmRelease of the dogu:

```bash
kubectl -n ecosystem delete helmrelease nexus
```

Flux removes the chart together with the failed jobs, the operator creates the HelmRelease again and the jobs start from the beginning.
This also restarts the `migrationJobStartTimeout`.

If the migration cannot be completed, restore the backup.

### What not to do

- Do not delete the secret or the claim `<dogu>-migration`.
  Until the migration is complete, the claim holds the only copy of the data.
  Without these resources, the operator no longer detects the migration and installs the dogu without data.
- Do not set `spec.doguApiVersion` back to `v2`.

## After the migration

- The operator sets the reclaim policy of the v2 volume to `Delete`, so the provisioner deletes the volume and its data.
  A volume without a dynamic provisioner ends up `Failed` and has to be deleted manually:
  ```bash
  kubectl get pv | grep nexus-migration
  ```
- The data of the dogu in shared dogus, e.g. its database in `postgresql`, is not removed by the operator.
- The dogu is a v3 dogu and cannot be turned back into a v2 dogu.
