# v3-Dogus installieren

Ein v3-Dogu wird wie ein v2-Dogu über eine Dogu-Ressource installiert.
Der Unterschied liegt in der API-Version der Ressource und im Feld `spec.doguApiVersion`, das festlegt, dass das Dogu als v3-Dogu installiert wird.
Für die Installation von v3-Dogus müssen im Cluster die folgenden Voraussetzungen erfüllt sein.

Die Beispiele gehen vom Namespace `ecosystem` aus und verwenden das Nexus-Dogu.

## Voraussetzungen

- **Flux** ist installiert, über `ecosystem-core` mit `flux.enabled: true`.
  Der Dogu-Operator nutzt Flux, um v3-Dogus auszurollen.
- **v3-Dogus sind im Dogu-Operator aktiviert**, siehe [v3-Dogus im Operator aktivieren](#v3-dogus-im-operator-aktivieren).
- **Secret `dogu-registry-v3`** enthält Endpoint und Zugangsdaten der Dogu-Registry v3, siehe [Secrets](#secrets).
- **Secret `ces-container-registries`** enthält einen Eintrag für die Registry, aus der die Charts geladen werden (für offizielle Dogus `registry.cloudogu.com`).

### v3-Dogus im Operator aktivieren

v3-Dogus werden nur verarbeitet, wenn der Helm-Value `controllerManager.env.doguV3Enabled` des Operators `true` ist.
Ist er nicht gesetzt, ignoriert der Operator v3-Dogus.
Im CES wird er über die Component-Ressource des Operators gesetzt:

```bash
kubectl -n ecosystem patch component k8s-dogu-operator --type=merge -p '
spec:
  valuesYamlOverwrite: |
    controllerManager:
      env:
        doguV3Enabled: true
'
```

> [!WARNING]
> Der Patch ersetzt den gesamten bisherigen Inhalt von `valuesYamlOverwrite`.
> Bestehende Overrides müssen deshalb im Patch enthalten sein.

### Secrets

v3-Dogus werden aus der Dogu-Registry v3 geladen.
Der Operator benötigt dafür eigene Zugangsdaten, getrennt von denen für v2-Dogus.

| Secret                     | Inhalt                                                                                 |
|----------------------------|----------------------------------------------------------------------------------------|
| `dogu-registry-v3`         | `endpoint`, `username`, `password`; optional `urlschema`, `insecureSkipVerify`          |
| `ces-container-registries` | Zugangsdaten der Container-Registry; wird vom CES-Setup angelegt                       |
| `dogu-v3-registry-cert`    | optional: eigenes CA-Zertifikat (`dogu-v3-registry-cert.pem`) für die Dogu-Registry v3 |

Der `endpoint` von `dogu-registry-v3` enthält den vollständigen Pfad der API, z. B. `https://dogu.cloudogu.com/api/v3/dogus`.
Wie das Secret angelegt wird, steht in [Konfiguration der Dogu-Registry](../configuring_the_dogu_registry_de.md).

`ces-container-registries` wird bereits von v2-Dogus verwendet.
Für v3-Dogus muss es zusätzlich Zugriff auf die Registry gewähren, in der die Charts der Dogus liegen.

> [!IMPORTANT]
> Der Operator liest `dogu-registry-v3` nur beim Start.
> Nach einer Änderung muss er neu gestartet werden:
> `kubectl -n ecosystem rollout restart deployment k8s-dogu-operator-controller-manager`

### Weitere Einstellungen des Operators

Die folgenden Helm-Values sind optional.
Sie legen fest, wie oft der Zustand von v3-Dogus geprüft wird und wie mit Fehlern umgegangen wird:

| Helm-Value (`controllerManager.env.`) | Default | Bedeutung                                                                   |
|---------------------------------------|---------|-----------------------------------------------------------------------------|
| `doguHelmReconciliationInterval`      | `30s`   | Intervall, in dem manuelle Abweichungen an den Ressourcen eines v3-Dogus zurückgesetzt werden. |
| `doguHelmRetryInterval`               | `5m`    | Wartezeit vor einem neuen Versuch, wenn Installation oder Upgrade fehlschlagen. |

Die Werte werden als Dauer angegeben, z. B. `20s`, `2m` oder `1h20m`.
Sie werden wie `doguV3Enabled` über die Component-Ressource des Operators gesetzt.

## Ein v3-Dogu installieren

Für die Installation wird eine Dogu-Ressource in der API-Version `k8s.cloudogu.com/v3beta1` mit `spec.doguApiVersion: v3` angelegt.
Das Beispiel installiert Nexus aus dem Dogu-Namespace `official`; `<version>` steht für eine als v3-Dogu verfügbare Version.

`nexus.yaml`:

```yaml
apiVersion: k8s.cloudogu.com/v3beta1
kind: Dogu
metadata:
  name: nexus
  labels:
    app: ces
    k8s.cloudogu.com/dogu.name: nexus
spec:
  name: nexus
  doguNamespace: official
  version: <version>
  doguApiVersion: v3
```

```bash
kubectl -n ecosystem apply -f nexus.yaml
```

| Feld                  | Bedeutung                                                                       |
|-----------------------|---------------------------------------------------------------------------------|
| `metadata.name`       | Name des Dogus; muss mit `spec.name` übereinstimmen.                            |
| `spec.name`           | Name des Dogus ohne Dogu-Namespace.                                             |
| `spec.doguNamespace`  | Namespace des Dogus in der Dogu-Registry, z. B. `official`.                     |
| `spec.version`        | Zu installierende Version.                                                      |
| `spec.doguApiVersion` | Muss `v3` sein; ohne das Feld wird das Dogu als v2-Dogu installiert.            |

Das Dogu kann direkt bei der Installation konfiguriert werden, indem `spec.values` oder `spec.mappedValues` in der Ressource mit angegeben werden.
Beides ist in [v3-Dogus konfigurieren](configuring_v3_dogus_de.md) beschrieben.

Hinweise:

- v3-Dogus immer über `dogus.v3beta1.k8s.cloudogu.com` lesen und bearbeiten.
  Ein einfaches `kubectl get dogu` nutzt die API-Version `v2` und zeigt v3-Felder wie `values` nicht.
- Eine Rückmigration von v3 nach v2 wird nicht unterstützt.
  Der Operator lehnt sie mit `Ready=False` und dem Reason `DowngradeNotAllowed` ab.
- Wird `spec.doguApiVersion: v3` an einem bereits installierten v2-Dogu gesetzt, wird es nach v3 migriert, siehe [Migration von v2](migrating_from_v2_to_v3_de.md).

## Installation prüfen

Den Stand der Installation zeigt die Übersicht der Dogu-Ressource:

```bash
kubectl -n ecosystem get dogus.v3beta1.k8s.cloudogu.com nexus
```

```text
NAME    SPEC VERSION   INSTALLED VERSION   APP VERSION   READY   HEALTHY   APIVERSION   STOPPED   PAUSE RECONCILIATION   AGE
nexus   <version>      <version>           3.x.y         True    True      v3                                          5m
```

Das Dogu ist installiert, wenn `READY` und `HEALTHY` beide `True` sind.
`READY` zeigt, ob die Installation abgeschlossen ist, `HEALTHY`, ob alle Pods des Dogus laufen und bereit sind.
Während der Installation steht `READY` auf `False`.

Bleibt eines davon auf `False`, nennen Conditions und Events die Ursache:

```bash
kubectl -n ecosystem get dogus.v3beta1.k8s.cloudogu.com nexus \
  -o jsonpath='{range .status.conditions[*]}{.type}={.status} ({.reason}): {.message}{"\n"}{end}'
kubectl -n ecosystem get events --field-selector involvedObject.kind=Dogu,involvedObject.name=nexus
```

Der Reason der Condition `Ready` beschreibt den aktuellen Schritt bzw. den Fehler, z. B. `Installing` oder `DownloadFailed`.
Alle Conditions und Events sind in [Conditions und Events von v3-Dogus](conditions_and_events_de.md) beschrieben.

## Häufige Fehler

| Symptom                                                   | Ursache / Lösung                                                                                   |
|-----------------------------------------------------------|----------------------------------------------------------------------------------------------------|
| Dogu-Ressource wird nicht verarbeitet, Operator-Log `dogu v3 feature flag is disabled` | v3-Dogus sind im Operator nicht aktiviert.                            |
| Dogu wird als v2-Dogu installiert                         | `spec.doguApiVersion: v3` fehlt.                                                                   |
| `Ready=False`, Reason `DownloadFailed`                    | `dogu-registry-v3`, Dogu-Namespace und Version prüfen; nach Änderung am Secret den Operator neu starten. |
| `Ready=False`, Reason `Unauthorized`                      | Eintrag für die Chart-Registry in `ces-container-registries` prüfen.                               |
| `Ready=False`, Reason `ChartNotFound`                     | Die Version existiert nicht als v3-Dogu.                                                           |
| `Ready=False`, Reason `SchemaInvalid` oder `RenderFailed` | Die Konfiguration passt nicht zum Dogu, siehe [Validierung](configuring_v3_dogus_de.md#validierung). |
| `Ready=False`, Reason `InstallFailed`                     | Die Installation ist fehlgeschlagen und wird nach `doguHelmRetryInterval` automatisch wiederholt; Details stehen in der Message der Condition. |
| `Ready=False`, Reason `WorkloadsNotReady`                 | Das Helm-Release ist ausgerollt, die Pods des Dogus sind nicht bereit; Pods und Logs prüfen.        |
| `Ready=True`, `Healthy=False`                             | Das Dogu war bereits bereit und ist danach ausgefallen, z. B. durch einen Pod-Absturz, oder es ist gestoppt (Reason `Stopped`); Pods und Logs prüfen. |
