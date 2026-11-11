# Migration von v2

Wird `spec.doguApiVersion: v3` an der Dogu-Ressource eines installierten v2-Dogus gesetzt, migriert der Operator es nach v3.
Er kopiert die Zugangsdaten der Service-Accounts in das Secret `<dogu>-migration`, überführt das Datenvolume in den
Claim `<dogu>-migration` und installiert das Chart mit `migrationJob.enabled: true`.
Die Migrations-Jobs des Charts übernehmen die Daten. Anschließend entfernt der Operator die Migrationsressourcen und
installiert das Chart regulär, wodurch die Migrations-Jobs wieder deaktiviert werden. Der Operator setzt die Reclaim-Policy
des Volumes des v2-Dogus auf `Delete`, sodass der Provisioner das Volume samt Daten unwiederbringlich löscht. Ein Volume
ohne dynamischen Provisioner endet im Status `Failed` und muss manuell entfernt werden.

Zwei Umgebungsvariablen begrenzen, wie lange der Operator auf die Migrations-Jobs wartet:
  - `MIGRATION_JOB_START_TIMEOUT` legt fest, wie lange nach dem Anlegen des HelmRelease auf den ersten Migrations-Job gewartet wird.
  - `MIGRATION_JOB_COMPLETION_TIMEOUT` legt fest, wie lange ein einzelner Migrations-Job laufen darf.

Beide werden wie die [Intervalle des Operators](installing_v3_dogus_de.md#weitere-einstellungen-des-operators) angegeben. Wird ein Timeout überschritten oder schlägt ein Job fehl, bleibt das
Dogu `Ready=False` mit dem Reason `MigrationFailed` und die Migrationsressourcen bleiben erhalten, auch das v2-Volume
mit der Reclaim-Policy `Retain`. Der Operator wiederholt die Migration nicht selbstständig. 
Das Start-Timeout zählt ab dem Anlegen des HelmRelease, daher muss das HelmRelease des Dogus nach Behebung der Ursache gelöscht werden, um das Timeout neu zu starten.

## Migrations-Jobs im Dogu-Chart

Ein Chart, das Daten migrieren muss, definiert seine Migrations-Jobs selbst und rendert sie nur, wenn
`migrationJob.enabled` `true` ist. Der Key `migrationJob` muss in der `values.yaml` des Charts existieren, sonst aktiviert
der Operator die Jobs nicht:

```yaml
migrationJob:
  enabled: false
```

Für die Jobs gelten folgende Konventionen:
  - Jeder Job trägt das Label `k8s.cloudogu.com/migration-job: "true"`. Der Operator ergänzt an allen Chart-Ressourcen
    das Label `k8s.cloudogu.com/dogu.name: <dogu>` und wartet, bis alle Jobs mit beiden Labels abgeschlossen sind.
  - Die Jobs bleiben nach ihrem Abschluss erhalten. Andernfalls findet der Operator keinen Job und die Migration schlägt fehl.
  - Ein Job, der ein geteiltes v2-Dogu wie `postgresql` erreichen muss, trägt in seinem Pod-Template die Labels
    `k8s.cloudogu.com/migration-postgresql: "true"` und `k8s.cloudogu.com/dogu.name: <dogu>`. Der Operator legt für
    jedes Dogu mit Zugangsdaten im Migration-Secret eine NetworkPolicy an, sofern NetworkPolicies aktiviert sind.
  - Die Jobs lesen die Zugangsdaten aus dem Secret `<dogu>-migration` (z. B. `postgresql_username`) und die Daten aus
    dem Claim `<dogu>-migration`.
  - Die Jobs sind idempotent, da sie nach einem fehlgeschlagenen Versuch erneut laufen.
  - Die Workloads des Dogus rendern `replicas: 0`, solange `migrationJob.enabled` `true` ist, damit die Anwendung nicht
    auf unvollständigen Daten startet.

Ein Chart ohne den Key `migrationJob` wird ohne Migrations-Jobs installiert. Existiert in diesem Fall der
Migration-Claim, schlägt die Migration fehl, um den Verlust der Daten zu verhindern.
