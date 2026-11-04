# Konfiguration für die Installation von v3-Dogus

## Flux-Abhängigkeiten

v3-Dogus werden als Helm-Charts mithilfe von zwei Flux-Controllern installiert: source-controller und helm-controller.
Beide müssen zusammen mit ihren CRDs installiert werden, damit v3-Dogus installiert werden können. Dies kann
mithilfe von ecosystem-core erfolgen, indem dessen Wert für `flux.enabled` auf `true` gesetzt wird. 
Neben Installations- und Upgrade-Vorgängen wird dadurch auch die Drift-Erkennung für Dogu-Installationen aktiviert, 
die manuelle Änderungen am zugehörigen Cluster-Status automatisch rückgängig macht.

## Konfiguration

Der Dogu-Operator benötigt einige zusätzliche Konfigurationen, um die Artefakte für v3-Dogus zu finden und abzurufen:

Das Secret `dogu-registry-v3` muss die URL und die Anmeldedaten für die v3-Dogu-Registry enthalten
(siehe [Konfiguration der Dogu-V3-Registry](configuring_the_dogu_registry_de.md)).

Dazu gibt es einige Konfigurationsoptionen, die per Umgebungsvariablen gesetzt werden, um festzulegen, 
wie die Helm-Installation gehandhabt wird:
  - `DOGU_HELM_RETRY_INTERVAL` steuert, wie lange vor einem erneuten Versuch gewartet werden soll, 
    falls die Helm-Installation oder das Upgrade fehlschlägt.
  - `DOGU_HELM_RECONCILIATION_INTERVAL` definiert das Intervall, in dem auf Abweichungen vom von Helm angeforderten Zustand geprüft wird,
    die dann automatisch korrigiert werden.

Beide Intervalle werden als Zahl mit einer Zeiteinheit angegeben, z. B. „20s“ (für 20 Sekunden), „2m“ (für 2 Minuten)
oder „1h“ (für 1 Stunde). Kombinationen wie „1h20m“ (für 1 Stunde und 20 Minuten) sind ebenfalls möglich.

## Stoppen und Starten

Mit `spec.stopped: true` an der Dogu-Ressource werden die Workloads gestoppt. Der Operator wartet auf die Suspendierung
des HelmRelease, skaliert Deployments und StatefulSets auf null und wartet, bis deren Pods verschwunden sind.
StatefulSet-PVCs bleiben erhalten. `status.stopped: true` bestätigt den abgeschlossenen Stopp.

Im gestoppten Zustand aktualisiert der Operator weiterhin Chart-Quellen und HelmRelease-Eingaben; Flux wendet diese
während der Suspendierung nicht an. Mit `spec.stopped: false` wird das Dogu wieder gestartet. Der Stoppstatus wird
nach erfolgreichem Deployment und erfolgreicher Health-Prüfung zurückgesetzt.

`spec.pauseReconciliation: true` hat Vorrang vor Start und Stopp: Reguläre Ressourcenänderungen werden übersprungen,
die Prüfungen von Quelle und Release können aber weiterhin `UpdatePending=True` setzen. Ein True für die aktuelle
Generation bleibt während Pause oder Stopp erhalten. Eine ältere Condition wird erst zurückgesetzt, wenn alle
relevanten Prüfungen ohne ausstehende Änderungen abgeschlossen sind. Eine unvollständige Prüfung lässt die Condition
unverändert und wird wiederholt.

## Migration von v2

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

Beide werden wie die obigen Intervalle angegeben. Wird ein Timeout überschritten oder schlägt ein Job fehl, bleibt das
Dogu `Ready=False` mit dem Reason `MigrationFailed` und die Migrationsressourcen bleiben erhalten, auch das v2-Volume
mit der Reclaim-Policy `Retain`. Der Operator wiederholt die Migration nicht selbstständig. 
Das Start-Timeout zählt ab dem Anlegen des HelmRelease, daher muss das HelmRelease des Dogus nach Behebung der Ursache gelöscht werden, um das Timeout neu zu starten.

### Migrations-Jobs im Dogu-Chart

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
