# Migrating from v2

Setting `spec.doguApiVersion: v3` on the dogu resource of an installed v2 dogu migrates it to v3.
The operator copies the service account credentials into the secret `<dogu>-migration`, moves the data volume to the
claim `<dogu>-migration` and installs the chart with `migrationJob.enabled: true`.
The chart's migration jobs take over the data. Afterward, the operator removes the migration resources and installs
the chart the regular way, which disables the migration jobs again. The operator sets the reclaim policy of the v2
dogu's volume to `Delete`, so the provisioner deletes the volume and its data irrecoverably. A volume without a dynamic
provisioner ends up `Failed` and has to be removed manually.

Two environment variables limit how long the operator waits for the migration jobs:
 - `MIGRATION_JOB_START_TIMEOUT` defines how long to wait for the first migration job after the HelmRelease was created.
 - `MIGRATION_JOB_COMPLETION_TIMEOUT` defines how long a single migration job may run.

Both are durations like the [intervals of the operator](installing_v3_dogus_en.md#further-operator-settings). If a timeout is exceeded or a job fails, the dogu stays `Ready=False` with
the reason `MigrationFailed` and the migration resources are kept, including the v2 volume with the
reclaim policy `Retain`. The operator does not retry the migration on its own.
The start timeout counts from the creation of the HelmRelease, so delete the HelmRelease of the dogu after fixing the
cause to restart the timeout.

## Migration jobs in the dogu chart

A chart that has to migrate data defines its migration jobs itself and renders them only if `migrationJob.enabled` is
`true`. The key `migrationJob` must exist in the chart's `values.yaml`, otherwise the operator does not enable the jobs:

```yaml
migrationJob:
  enabled: false
```

The jobs follow these conventions:
 - Each job carries the label `k8s.cloudogu.com/migration-job: "true"`. The operator adds the label
   `k8s.cloudogu.com/dogu.name: <dogu>` to all chart resources and waits for all jobs with both labels to complete.
 - The jobs remain after completion. Otherwise, the operator finds no job and fails the migration.
 - A job that has to reach a shared v2 dogu, e.g. `postgresql`, carries the labels
   `k8s.cloudogu.com/migration-postgresql: "true"` and `k8s.cloudogu.com/dogu.name: <dogu>` in its pod template.
   The operator creates a network policy for each dogu with credentials in the migration secret, if network policies
   are enabled.
 - The jobs read the credentials from the secret `<dogu>-migration` (e.g. `postgresql_username`) and the data from
   the claim `<dogu>-migration`.
 - The jobs are idempotent, because they run again after a failed attempt.
 - The dogu's workloads render `replicas: 0` while `migrationJob.enabled` is `true`, so the application does not start
   on incomplete data.

A chart without the key `migrationJob` is installed without migration jobs. If the migration claim exists in that case,
the migration fails to prevent the loss of the data.
