# Continuous Reconciliation

Dogus get continuously reconciled until the desired state of the dogu resource is reached.

## Restarts on config change

This concerns restarts when the dogu's configuration changes (located in the ConfigMap and Secret named
`<dogu-name>-config`).
Changes in the global config trigger a restart of all dogus (located in the ConfigMap named `global-config`).

## Pause reconciliation

If continuous reconciliation and automatic restarts are not desired in specific cases, it is possible to temporarily
pause reconciliation via the `spec.pauseReconciliation` flag of the dogu resource.

This will prevent **ALL** changes to the dogu resource from being applied (except `spec.pauseReconciliation`).
Validation will still be executed.

**WARNING**: Only enable this option temporarily, e.g. for debugging purposes or if you want to update the dogu and
change the config simultaneously without a restart in between.
