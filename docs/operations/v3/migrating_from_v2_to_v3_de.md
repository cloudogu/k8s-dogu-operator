# Migration von v2

Ein installiertes v2-Dogu wird nach v3 migriert, indem an seiner Dogu-Ressource `spec.doguApiVersion: v3` gesetzt wird.
Der Dogu-Operator erkennt die Migration selbst und führt sie vollständig durch.
Übernommen werden die Daten des Volumes und die Zugangsdaten der Service-Accounts des Dogus.
Die Konfiguration des v2-Dogus wird nicht übernommen.

> [!WARNING]
> Es gibt keinen Weg zurück zu v2.
> Nach einer erfolgreichen Migration löscht der Operator das Volume des v2-Dogus.
> Vor der Migration sollte ein Backup erstellt werden.

Die Beispiele gehen vom Namespace `ecosystem` aus und verwenden das Nexus-Dogu.

## Ablauf

1. Der Operator prüft, ob das v2-Dogu migriert werden kann, siehe [Voraussetzungen](#voraussetzungen).
2. Er lädt das Chart der Zielversion und validiert es zusammen mit den Values.
   Schlägt das fehl, bleibt das v2-Dogu unverändert.
3. Er kopiert die Zugangsdaten der Service-Accounts aus der sensiblen Konfiguration des v2-Dogus in das Secret
   `<dogu>-migration`.
4. Er skaliert das v2-Dogu auf 0. **Ab hier beginnt die Downtime des Dogus.**
   Das Volume des v2-Dogus bleibt erhalten und wird an den neuen Claim `<dogu>-migration` gebunden.
5. Er löscht die übrigen Ressourcen des v2-Dogus: Konfiguration, sensible Konfiguration, Deployment, Services,
   NetworkPolicies sowie die Ressourcen `AuthRegistration`, `Exposition` und `WarpMenuEntry`.
6. Bringt das Chart Migrations-Jobs mit, installiert der Operator das Chart mit aktivierten Migrations-Jobs.
   Die Jobs übernehmen die Daten aus dem Claim `<dogu>-migration` und bei Bedarf aus geteilten v2-Dogus wie `postgresql`. 
   Die Anwendung des Dogus startet in dieser Zeit nicht.
   Der Operator wartet, bis alle Migrations-Jobs abgeschlossen sind.
7. Er entfernt das Secret und den Claim `<dogu>-migration`.
   Das Volume des v2-Dogus wird samt Daten gelöscht, siehe [Nach der Migration](#nach-der-migration).
8. Er installiert das Chart regulär. Das Dogu ist danach ein gewöhnliches v3-Dogu.

Eine unterbrochene Migration, z. B. durch einen Neustart des Operators, wird an der Stelle fortgesetzt, an der sie stehen geblieben ist.

Hat das v2-Dogu ein Volume, muss das Chart der Zielversion Migrations-Jobs mitbringen.
Andernfalls schlägt die Migration mit `MigrationFailed` fehl, bevor die Daten verloren gehen.

## Vorbereitung

### Voraussetzungen

- v3-Dogus sind im Operator aktiviert, siehe [v3-Dogus installieren](installing_v3_dogus_de.md#voraussetzungen).
- Die Zielversion existiert als v3-Dogu und unterstützt die Migration von v2, siehe Release Notes des Dogus.
- Das v2-Dogu ist installiert und es läuft keine Operation.
- Support-Modus und Export-Modus sind nicht aktiv und das Volume des Dogus wird nicht vergrößert.

Ist einer der letzten beiden Punkte nicht erfüllt, startet der Operator die Migration nicht.
Das Dogu bleibt dann `Ready=False` mit dem Reason `MigrationPending` und der Message `The migration to v3 is blocked by the v2 dogu: <Grund>`.
Der Operator prüft jede Minute erneut und startet die Migration, sobald die Ursache behoben ist.

### Konfiguration übertragen

Der Operator übernimmt die Konfiguration des v2-Dogus nicht.
Konfiguration und sensible Konfiguration des v2-Dogus werden während der Migration gelöscht.
Nur die Zugangsdaten der Service-Accounts bleiben erhalten.

Vor der Migration muss die Konfiguration des v2-Dogus ausgelesen und in `spec.values` oder `spec.mappedValues`
übertragen werden, siehe [v3-Dogus konfigurieren](configuring_v3_dogus_de.md):

```bash
kubectl -n ecosystem get configmap nexus-config -o jsonpath='{.data.config\.yaml}'
kubectl -n ecosystem get secret nexus-config -o jsonpath='{.data.config\.yaml}' | base64 -d
```

Die Values werden in derselben Änderung wie `spec.doguApiVersion` gesetzt, siehe [Migration starten](#migration-starten).
So startet das Dogu direkt nach der Migration mit seiner Konfiguration.

### Backup

Vor der Migration muss das Dogu gesichert werden, einschließlich seiner Daten in geteilten Dogus wie `postgresql`.
Lässt sich die Migration nicht abschließen, ist das Backup der einzige Weg zurück zu einem funktionierenden Stand.

### Speicherplatz

Während der Migration existiert das Volume des v2-Dogus neben den neuen Volumes des v3-Dogus.
Bis zum Abschluss der Migration wird daher die doppelte Speicherkapazität benötigt.

Die neuen Volumes müssen mindestens so groß sein wie das Volume des v2-Dogus, damit die Daten hineinpassen.
Ihre Größe wird über die Values des Dogus festgelegt; der Operator prüft sie nicht.

### Timeouts der Migrations-Jobs

Der Operator wartet nur begrenzt auf die Migrations-Jobs:

| Helm-Value (`controllerManager.env.`) | Default | Bedeutung                                                                        |
|---------------------------------------|---------|----------------------------------------------------------------------------------|
| `migrationJobStartTimeout`            | `5m`    | Zeit vom Anlegen des HelmRelease, bis der erste Migrations-Job erscheint.         |
| `migrationJobCompletionTimeout`       | `1h`    | Zeit, die ein einzelner Migrations-Job laufen darf.                              |

Bei großen Datenmengen sollte `migrationJobCompletionTimeout` vor der Migration erhöht werden.
Die Werte werden wie die [weiteren Einstellungen des Operators](installing_v3_dogus_de.md#weitere-einstellungen-des-operators) über die Component-Ressource des Operators gesetzt.

## Migration starten

Die Migration startet, sobald die Dogu-Ressource `spec.doguApiVersion: v3` enthält.
Version des v3-Dogus und Konfiguration werden in derselben Änderung gesetzt:

```bash
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge -p '
spec:
  doguApiVersion: v3
  version: <version>
  values:
    # Konfiguration des Dogus
'
```

## Migration verfolgen

Während der Migration ist das Dogu `Ready=False` mit dem Reason `MigrationPending`.
Die Message der Condition zeigt den aktuellen Schritt:

```bash
kubectl -n ecosystem get dogus.v3beta1.k8s.cloudogu.com nexus \
  -o jsonpath='{range .status.conditions[?(@.type=="Ready")]}{.reason}: {.message}{"\n"}{end}'
```

```text
MigrationPending: Waiting for 1 of 2 migration jobs to complete
```

Die Events zeigen den Verlauf der Migration mit Zeitstempeln:

```bash
kubectl -n ecosystem get events --field-selector involvedObject.kind=Dogu,involvedObject.name=nexus
```

| Typ     | Reason               | Message (Beispiele)                                                                             |
|---------|----------------------|-------------------------------------------------------------------------------------------------|
| Normal  | `MigrationPending`   | `Migration secret "nexus-migration" created`                                                    |
| Normal  | `MigrationPending`   | `Prepared the dogu pvc for the v3 migration`                                                    |
| Normal  | `MigrationPending`   | `Initiated deletion of 7 resources of the v2 installation`                                      |
| Normal  | `MigrationPending`   | `Migration network policy "nexus-migration-postgresql" created`                                 |
| Normal  | `MigrationPending`   | `Waiting for 1 of 2 migration jobs to complete`                                                 |
| Normal  | `MigrationPending`   | `All 2 migration jobs completed`                                                                |
| Normal  | `MigrationPending`   | `Migration resources removed`                                                                   |
| Normal  | `ReconcileSucceeded` | `Dogu reconciliation completed successfully`                                                    |
| Warning | `MigrationPending`   | Ein Schritt ist vorübergehend fehlgeschlagen; der Operator wiederholt ihn automatisch.          |
| Warning | `MigrationFailed`    | Die Migration wurde abgebrochen, siehe [Fehlgeschlagene Migration](#fehlgeschlagene-migration). |

Die Migrations-Jobs und ihre Logs:

```bash
kubectl -n ecosystem get jobs -l k8s.cloudogu.com/migration-job=true,k8s.cloudogu.com/dogu.name=nexus
kubectl -n ecosystem logs job/<job>
```

Die Migration ist abgeschlossen, wenn das Event `Migration resources removed` erscheint.
Danach startet das Dogu wie bei einer Installation.
Es ist betriebsbereit, wenn es `Healthy=True` und `Ready=True` ist.

## Fehlgeschlagene Migration

Lässt sich die Migration nicht abschließen, bleibt das Dogu `Ready=False` mit dem Reason `MigrationFailed`.
Zusätzlich gibt es ein Warning-Event mit derselben Message.
Der Operator wiederholt die Migration nicht selbstständig.

Die Daten gehen dabei nicht verloren: Secret und Claim `<dogu>-migration` sowie das Volume des v2-Dogus bleiben erhalten.

| Message                                                                                    | Ursache / Lösung                                                                                                                                                                                                                           |
|--------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `Migration job "<job>" failed: ...`                                                        | Ein Migrations-Job ist fehlgeschlagen. Logs des Jobs prüfen, Ursache beheben und [Migrations-Jobs neu starten](#migrations-jobs-neu-starten).                                                                                              |
| `Migration job "<job>" did not complete within <Dauer>`                                    | Ein Migrations-Job lief länger als `migrationJobCompletionTimeout`. Timeout erhöhen und [Migrations-Jobs neu starten](#migrations-jobs-neu-starten).                                                                                       |
| `No migration job of dogu "<dogu>" found within <Dauer>`                                   | Innerhalb von `migrationJobStartTimeout` ist kein Migrations-Job erschienen. HelmRelease des Dogus prüfen (`kubectl -n ecosystem get helmrelease nexus`), Ursache beheben und [Migrations-Jobs neu starten](#migrations-jobs-neu-starten). |
| `the chart ships no migration job, so the data of the migration volume claim ...`          | Die Zielversion unterstützt die Migration nicht. `spec.version` auf eine Version setzen, die sie unterstützt; die Migration läuft dann automatisch weiter.                                                                                 |
| `failed to parse sensitive config: ...`, `... both result in the migration secret key ...` | Die sensible Konfiguration des v2-Dogus (`<dogu>-config`) ist nicht lesbar. Secret korrigieren; die Migration läuft dann automatisch weiter. Das v2-Dogu läuft in diesem Fall noch.                                                        |

### Migrations-Jobs neu starten

Die Migrations-Jobs werden erneut ausgeführt, indem das HelmRelease des Dogus gelöscht wird:

```bash
kubectl -n ecosystem delete helmrelease nexus
```

Flux entfernt das Chart samt der fehlgeschlagenen Jobs, der Operator legt das HelmRelease neu an und die Jobs starten von vorn.
Dadurch beginnt auch `migrationJobStartTimeout` neu.

Lässt sich die Migration nicht abschließen, muss das Backup zurückgespielt werden.

### Wichtige Hinweise

- Secret und Claim `<dogu>-migration` nicht löschen.
  Bis zum Abschluss der Migration enthält der Claim die einzige Kopie der Daten.
  Ohne diese Ressourcen erkennt der Operator die Migration nicht mehr und installiert das Dogu ohne Daten.
- `spec.doguApiVersion` nicht auf `v2` zurücksetzen.

## Nach der Migration

- Der Operator setzt die Reclaim-Policy des v2-Volumes auf `Delete`, sodass der Provisioner das Volume samt Daten löscht.
  Ein Volume ohne dynamischen Provisioner endet im Status `Failed` und muss manuell gelöscht werden:
  ```bash
  kubectl get pv | grep nexus-migration
  ```
- Die Daten des Dogus in geteilten Dogus, z. B. seine Datenbank in `postgresql`, entfernt der Operator nicht.
- Das Dogu ist ein v3-Dogu und kann nicht wieder zu einem v2-Dogu werden.
