# Conditions und Events von v3-Dogus

Der Zustand eines v3-Dogus steht in den Conditions der Dogu-Ressource.
Jede Condition beschreibt einen Aspekt, z. B. ob die Installation abgeschlossen ist (`Ready`) oder ob die Pods laufen (`Healthy`).
Der Status ist `True` oder `False`, der Reason nennt den Grund und die Message liefert Details.
Bei Erfolg ist der Reason immer `Succeeded`.

Zusätzlich schreibt der Dogu-Operator Events an die Dogu-Ressource, z. B. wenn ein Vorgang abgeschlossen ist oder fehlschlägt.

```bash
# Conditions
kubectl -n ecosystem get dogus.v3beta1.k8s.cloudogu.com <dogu> \
  -o jsonpath='{range .status.conditions[*]}{.type}={.status} ({.reason}): {.message}{"\n"}{end}'

# Events
kubectl -n ecosystem get events --field-selector involvedObject.kind=Dogu,involvedObject.name=<dogu>
```

## Conditions

### `Ready`

`Ready` beschreibt den Gesamtzustand der Dogu-Ressource.
Sie ist `True`, wenn der gewünschte Zustand der Dogu-Ressource vollständig umgesetzt ist.
Bei `False` nennt der Reason den laufenden Vorgang oder den aufgetretenen Fehler:

| Reason                                                                   | Bedeutung                                                                             |
|--------------------------------------------------------------------------|---------------------------------------------------------------------------------------|
| `Installing`, `Upgrading`, `Starting`, `Stopping`, `ResizingPVC`         | Ein Vorgang läuft; kein Handlungsbedarf.                                              |
| `WorkloadsNotReady`                                                      | Das Dogu ist ausgerollt, aber noch nicht alle Pods sind bereit.                       |
| `ReconciliationPaused`                                                   | `spec.pauseReconciliation` ist gesetzt.                                               |
| `DownloadFailed`                                                         | Das Dogu konnte nicht aus der Dogu-Registry geladen werden.                           |
| `Unauthorized`                                                           | Kein Zugriff auf die Chart-Registry (`ces-container-registries`).                     |
| `ChartNotFound`                                                          | Die Version existiert nicht als v3-Dogu.                                              |
| `SchemaInvalid`, `RenderFailed`, `VolumeShrink`, `StorageClassImmutable` | Ungültige Konfiguration, siehe [Validierung](configuring_v3_dogus_de.md#validierung). |
| `InstallFailed`, `UpgradeFailed`                                         | Installation bzw. Upgrade ist fehlgeschlagen; wird automatisch wiederholt.            |
| `MigrationPending`, `MigrationFailed`                                    | Siehe [Migration von v2](migrating_from_v2_to_v3_de.md).                              |
| `Deleting`, `DeletionFailed`, `DeletionStalled`                          | Das Dogu wird gelöscht bzw. das Löschen ist fehlgeschlagen oder blockiert.            |
| `DowngradeNotAllowed`                                                    | Eine Rückmigration von v3 nach v2 wird nicht unterstützt.                             |
| `StepFailed`, `ReconciliationAborted`                                    | Allgemeiner Fehler; Message und Operator-Log nennen die Ursache.                      |

Schlägt ein Vorgang fehl, läuft das Dogu in der Regel mit dem bisherigen Stand weiter.
Nach Korrektur der Ursache setzt der Operator den Vorgang automatisch fort.

### Weitere Conditions

| Condition                 | `True` bedeutet                                            | Reasons bei `False` bzw. Details                                                 |
|---------------------------|------------------------------------------------------------|----------------------------------------------------------------------------------|
| `Healthy`                 | Alle Pods des Dogus sind bereit.                           | `WorkloadsNotReady`, `Stopped`                                                   |
| `ChartAvailable`          | Das Chart des Dogus wurde geladen.                         | `DownloadFailed`, `Unauthorized`, `ChartNotFound`                                |
| `Valid`                   | Die Konfiguration ist gültig.                              | `SchemaInvalid`, `RenderFailed`, `VolumeShrink`, `StorageClassImmutable`         |
| `SchemaValidationSkipped` | `spec.skipSchemaValidation` ist gesetzt.                   | `SchemaValidationEnabled`                                                        |
| `Stopped`                 | Das Dogu ist vollständig gestoppt.                         | `PartiallyStopped` (Stoppen läuft), `NotStopped`                                 |
| `Suspended`               | Änderungen werden nicht ausgerollt.                        | Reasons bei `True`: `DoguStopped`, `ReconciliationPaused`, `PVCResizeInProgress` |
| `UpdatePending`           | Es gibt Änderungen, die erst nach Start bzw. Pause wirken. | Reasons: `Stopped`, `ReconciliationPaused`                                       |
| `ExportModeActive`        | –                                                          | Für v3-Dogus noch nicht umgesetzt (`NotImplemented`)                             |

`Healthy` wird unabhängig von `Ready` geprüft.
Ein Dogu kann also `Ready=True` und gleichzeitig `Healthy=False` sein, z. B. wenn es gestoppt ist oder ein Pod abstürzt.

## Events

Events protokollieren Vorgänge und Zustandswechsel des Dogus und dienen vor allem der nachträglichen Fehleranalyse.

| Typ     | Reason                         | Auslöser                                                    |
|---------|--------------------------------|-------------------------------------------------------------|
| Normal  | `ReconcileSucceeded`           | Der gewünschte Zustand ist umgesetzt.                       |
| Normal  | `ChartAvailable`               | Das Chart wurde geladen bzw. die Installation aktualisiert. |
| Warning | Reason von `Ready`             | Ein Vorgang ist abgebrochen, z. B. `SchemaInvalid`.         |
| Normal  | `Succeeded`                    | `Healthy` wechselt auf `True`.                              |
| Warning | `WorkloadsNotReady`, `Stopped` | `Healthy` wechselt auf `False`.                             |
| Normal  | `UpdatePending`                | Eine Änderung wartet auf das Ende der Pause.                |
| Normal  | `ResizingPVC`                  | Ein Volume wird vergrößert.                                 |

Bei automatischen Wiederholungen und während einer Pause entstehen keine Warning-Events.
Der aktuelle Zustand steht immer in den Conditions.
Events zu Neustarts stehen an der jeweiligen `DoguRestart`-Ressource, nicht am Dogu.
