# v3-Dogus konfigurieren

v3-Dogus werden über Helm-Values konfiguriert.
Sie ersetzen die Dogu-Config-Keys, über die v2-Dogus konfiguriert wurden.
Administratoren setzen die Values an der Dogu-Ressource, entweder direkt in `spec.values` oder über generische Schlüssel in `spec.mappedValues`.
Zusätzlich erhält jedes v3-Dogu die globale Konfiguration des CES.
Der Dogu-Operator führt alle Quellen zusammen und rollt das Ergebnis aus.

Die Beispiele gehen vom Namespace `ecosystem` aus und verwenden das Nexus-Dogu.

## Verfügbare Values

Die verfügbaren Values legt das jeweilige Dogu fest.
Sie stehen in der Dokumentation des Dogus und in den folgenden Dateien seines Helm-Charts:

| Datei                       | Inhalt                                                             |
|-----------------------------|--------------------------------------------------------------------|
| `values.yaml`               | Alle Values mit ihren Defaults                                     |
| `values.schema.json`        | Optional: erlaubte Werte und Typen, gegen die geprüft wird         |
| `dogu-values-metadata.yaml` | Optional: die generischen Schlüssel für `spec.mappedValues`        |

## Priorisierung der Values

Der Operator setzt die Values aus mehreren Quellen zusammen.
Höhere Ränge überschreiben niedrigere:

| Rang | Quelle                   | Beschreibung                                                                 |
|------|--------------------------|------------------------------------------------------------------------------|
| 0    | `values.yaml` des Charts | Defaults des Dogus                                                           |
| 1    | `global.cesConfig`       | Globale Konfiguration des CES, siehe [Globale Konfiguration](#globale-konfiguration) |
| 2    | Runtime-Patches          | Werte, die das Dogu aus der globalen Konfiguration ableitet, z. B. eine URL aus der FQDN |
| 3    | `spec.values`            | Beliebige Overrides der Values                                               |
| 4    | `spec.mappedValues`      | Generische Schlüssel, die das Dogu auf seine Values abbildet                 |

Beim Zusammenführen werden Maps rekursiv gemergt.
Listen und einfache Werte werden dagegen vollständig durch die Quelle mit dem höheren Rang ersetzt.
Ein Override einer Liste muss deshalb immer die vollständige Liste enthalten.

## `spec.values`

`spec.values` nimmt beliebige Overrides der Values des Dogus auf.
Es müssen nur die Werte angegeben werden, die vom Default abweichen; alle anderen behalten ihren Default.
Die Pfade entsprechen der Struktur der `values.yaml` des Dogus.

Das folgende Beispiel dient nur zur Veranschaulichung, die tatsächlichen Pfade hängen vom Dogu ab:

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
  values:
    nexus:
      configuration:
        normal:
          database:
            maxConnections: "50"
```

An einer bestehenden Dogu-Ressource lassen sich einzelne Werte per Merge-Patch ändern:

```bash
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge -p '
spec:
  values:
    nexus:
      configuration:
        normal:
          database:
            maxConnections: "50"
'
```

### Einen Wert zurücksetzen

Um wieder den Default des Dogus zu verwenden, wird der Override aus der Dogu-Ressource entfernt.
Bei einem Merge-Patch geschieht das, indem der Wert auf `null` gesetzt wird:

```bash
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge -p '
spec:
  values:
    nexus:
      configuration:
        normal:
          database:
            maxConnections: null
'
```

Der Merge-Patch löscht den Schlüssel aus `spec.values`, danach gilt wieder der Default.

> [!WARNING]
> Ein `null`, das in `spec.values` stehen bleibt (z. B. per `kubectl apply` einer YAML-Datei mit `maxConnections: null`), stellt den Default nicht wieder her.
> Es wird an Helm weitergegeben und entfernt den Schlüssel auch aus den Defaults.
> Werte deshalb entfernen, statt sie auf `null` zu setzen.

## `spec.mappedValues`

Mapped Values sind generische Schlüssel wie `mainLogLevel`, die für alle Dogus einheitlich gesetzt werden, unabhängig von der Struktur der Values des jeweiligen Dogus.
Das Dogu legt in seiner `dogu-values-metadata.yaml` fest, auf welche seiner Values ein Schlüssel wirkt und welche Werte erlaubt sind.

```yaml
spec:
  mappedValues:
    mainLogLevel: debug
```

- `spec.mappedValues` hat den höchsten Rang und überschreibt auch `spec.values`.
- Schlüssel, die das Dogu nicht kennt, werden ohne Fehlermeldung ignoriert.
  Fehlerhafte Schlüssel werden daher nicht gemeldet und bleiben wirkungslos.

### Log-Level

Das Log-Level eines Dogus wird über den Schlüssel `mainLogLevel` gesetzt.
Er ersetzt den Dogu-Config-Key `logging/root` von v2-Dogus.
Üblich sind die Werte `debug`, `info`, `warn` und `error`; maßgeblich ist die `dogu-values-metadata.yaml` des Dogus.
Das Dogu übersetzt den Wert in die Schreibweise, die seine Anwendung erwartet, z. B. `debug` in `DEBUG`.

```bash
# Log-Level setzen
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge \
  -p '{"spec":{"mappedValues":{"mainLogLevel":"debug"}}}'

# zurück auf den Default des Dogus
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge \
  -p '{"spec":{"mappedValues":{"mainLogLevel":null}}}'
```

## Globale Konfiguration

Die globale Konfiguration des CES (ConfigMap `global-config`) steht jedem v3-Dogu unter `global.cesConfig` zur Verfügung.
Das Dogu kann diese Werte direkt verwenden oder daraus eigene Values ableiten, z. B. Adressen aus der FQDN.

- Eine Änderung an der globalen Konfiguration wird sofort an alle v3-Dogus weitergegeben.
- Ob sie im Dogu wirkt und ob dafür ein Neustart nötig ist, hängt vom Dogu ab.
- Gestoppte oder pausierte Dogus übernehmen die Änderung erst nach dem Start bzw. dem Ende der Pause.
  Bis dahin zeigen sie `UpdatePending=True`.

## Validierung

Bevor der Operator eine Änderung ausrollt, prüft er die zusammengesetzten Values:

1. gegen die `values.schema.json` des Dogus, sofern vorhanden,
2. durch probeweises Rendern des Helm-Charts,
3. auf unzulässige Änderungen an Volumes.

Schlägt eine Prüfung fehl, wird nichts ausgerollt und das Dogu läuft mit der bisherigen Konfiguration weiter.
Die Ursache steht als Reason an den Conditions `Valid` und `Ready`, zusätzlich gibt es ein Warning-Event:

| Reason                  | Ursache                                                   |
|-------------------------|-----------------------------------------------------------|
| `SchemaInvalid`         | Die Values verletzen die `values.schema.json` des Dogus.  |
| `RenderFailed`          | Das Helm-Chart lässt sich mit den Values nicht rendern.   |
| `VolumeShrink`          | Ein Volume würde verkleinert.                             |
| `StorageClassImmutable` | Die StorageClass eines Volumes würde geändert.            |

### Schema-Prüfung abschalten

Mit `spec.skipSchemaValidation: true` wird die Prüfung gegen die `values.schema.json` abgeschaltet.
Das probeweise Rendern und die Prüfung der Volumes finden weiterhin statt.
Die Condition `SchemaValidationSkipped` ist dann `True`.

> [!WARNING]
> Ohne Schema-Prüfung werden ungültige Werte erst beim Rollout oder zur Laufzeit des Dogus erkannt.

## Neustart bei Konfigurationsänderungen

Der Dogu-Operator startet bei Konfigurationsänderungen keine Pods neu, sondern rollt nur die neuen Values aus.
Ob eine Änderung einen Neustart auslöst, legt das Dogu fest:
Manche Einstellungen führen automatisch zu einem Neustart der betroffenen Pods, andere liest das Dogu zur Laufzeit ein oder erst beim nächsten Start.
Welche Einstellungen einen Neustart auslösen, steht in der Dokumentation des Dogus.

Ein manueller Neustart ist jederzeit möglich, siehe [Neu starten](stopping_starting_and_restarting_v3_dogus_de.md#neu-starten).

## Nicht unterstützte Änderungen

- **Ressourcen des Dogus direkt ändern** (ConfigMaps, StatefulSets, ...):
  Solche Änderungen werden spätestens nach `doguHelmReconciliationInterval` zurückgesetzt.
  Konfiguriert wird ausschließlich über die Dogu-Ressource.
- **Die Dogu-Ressource über die API-Version `v2` bearbeiten** (`kubectl edit dogu`):
  `values` und `mappedValues` sind dort nicht sichtbar.
  v3-Dogus deshalb immer über `dogus.v3beta1.k8s.cloudogu.com` bearbeiten.

## Rollout einer Änderung prüfen

```bash
kubectl -n ecosystem get dogus.v3beta1.k8s.cloudogu.com nexus
```

Sobald `READY` und `HEALTHY` wieder `True` sind, ist die Änderung ausgerollt.
Bleibt `READY` auf `False`, nennen Conditions und Events die Ursache, siehe [Conditions und Events von v3-Dogus](conditions_and_events_de.md).
