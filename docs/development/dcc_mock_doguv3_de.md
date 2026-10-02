# Mocking des DCC für die Entwicklung von Dogu V3

Bei der Entwicklung des Dogu-V3-Codes muss der Dogu-Operator mit einer Dogu-Registry kommunizieren, 
die V3-Dogu-Deskriptoren bereitstellt, sowie mit einer OCI-Registry für die Helm-Charts der Dogus.

Dieser Mock ersetzt die Dogu-Registry des DCC durch einen Nginx-Pod, sodass die V3 Fetch- und Render-Pfade (sowie deren Fehlerfälle) 
anhand einer vollständig lokalen, skriptgesteuerten Testumgebung getestet werden können.

Die OCI-/Helm-Registry wird **nicht** simuliert – der Operator ruft weiterhin das tatsächliche Helm-Chart von `registry.cloudogu.com` ab. 
Ein Chart für die referenzierte Version ist dort bereits vorhanden (siehe [Das Mock synchron halten](#das-mock-chart-synchron-halten)).

Alle Dateien des Mocks befinden sich unter [`dev/mock-dcc-v3/`](../../dev/mock-dcc-v3) und die Entrypoints
sind zwei Make-Targets: `mock-dcc-v3` und `mock-dcc-v3-clean`.

## Voraussetzungen

Diese Konfiguration wurde getestet mit:

- Einem lokalen **k3d**-Cluster.
- Dem **k8s-dogu-operator**, der im Cluster läuft und bei dem das V3-Feature-Flag aktiviert ist: `controllerManager.env.doguV3Enabled: true` [`k8s/helm/values.yaml`](../../k8s/helm/values.yaml).
- **ecosystem-core** mit `flux.enabled=true` installiert. V3-Dogus werden über Flux als Helm-Charts installiert, daher muss Flux im Cluster vorhanden sein.

Die Targets laufen in dem Namespace, der durch `NAMESPACE` in der `.env`-Datei angegeben ist (standardmäßig `ecosystem`).

## How it works

Der Operator ermittelt den Endpunkt der Dogu-Registry anhand des Secrets `dogu-registry-v3`.
Der Mock leitet diesen Endpunkt lediglich auf einen Nginx-Pod innerhalb des Clusters
(`test-dccv3`) um, der statische Deskriptor-Dateien bereitstellt. Jeder Deskriptor teilt dem
Operator mit, welches Helm-Chart (`Chart`) und welche Chart-Version (`Version`)
aus der echten OCI-Registry abgerufen werden sollen.

Der Nginx-Pod bindet die Configmap `test-v3-dogus` unter
`/usr/share/nginx/html/api/v3/dogus/internal/nexus` ein, sodass die Schlüssel der Configmap
zu den Versionspfadsegmenten werden, die Nginx bereitstellt.

## Was wird bereitgestellt?

Dateien unter [`dev/mock-dcc-v3/`](../../dev/mock-dcc-v3):

| Datei | Zweck |
| --- | --- |
| `nexus.json` | Deskriptor für den **Erfolgsfall** (gültiges Chart + veröffentlichte Version). |
| `nexus-not-found.json` | Deskriptor für die **helm-404** (gültiges Chart, Version nicht veröffentlicht). |
| `dcc.yaml` | Dar Mock selbst: nginx `Pod` + `Service` + `NetworkPolicy` (`test-dccv3`). |
| `dogu-success.yaml` | `Dogu`-CR für das Erfolgsszenario. |
| `dogu-404-helm-registry.yaml` | `Dogu`-CR für das „helm-404“-Szenario. |
| `dogu-404-dogu-registry.yaml` | `Dogu`-CR für das „dogu-registry-404“-Szenario. |

Die Deskriptor-Dateien werden als `test-v3-dogus`-Configmap
im Cluster veröffentlicht, sortiert nach Version. Die `Dogu`-CRs werden von den Make-Targets **nicht** angewendet –
je nach Szenario müssen sie manuell im Cluster deployt werden (siehe [Verwendung](#verwendung)).

## Verwendung

Setup des Mock (idempotent – kann nach der Bearbeitung der Deskriptoren sicher erneut ausgeführt werden):

```bash
make mock-dcc-v3
```

Anschließend jeweils **ein** Szenario anwenden:

```bash
kubectl -n ecosystem apply -f dev/mock-dcc-v3/dogu-success.yaml
kubectl -n ecosystem apply -f dev/mock-dcc-v3/dogu-404-helm-registry.yaml
kubectl -n ecosystem apply -f dev/mock-dcc-v3/dogu-404-dogu-registry.yaml
```

Beobachte die Operator-Logs bzw. den Status der `Dogu`-Ressource, um das Ergebnis zu überprüfen.

Entfernen des Mocks und Wiederherstellung der Registry:

```bash
make mock-dcc-v3-clean
```

Dadurch werden der Nginx-Pod, der Service, die Networkpolicies und die ConfigMap gelöscht, anschließend wird
der Endpunkt `dogu-registry-v3` aus der Datei `.endpoint.bak` wiederhergestellt (oder das Secret
vollständig gelöscht, falls das Backup den Eintrag `__ABSENT__` enthält) und schließlich das Backup
entfernt. Wird kein Backup gefunden, wird eine Warnung ausgegeben und das Secret bleibt unverändert.

## Das Mock-Chart synchron halten

Das Erfolgsszenario hängt davon ab, dass ein **echtes** Helm-Chart im OCI-Register
veröffentlicht wird. Ein Chart für die referenzierte Version
(`…/charts/nexus/dogu-v3-develop:3.86.2-6`) existiert bereits, sodass der Erfolgsfall
von Haus aus funktioniert.

Wenn sich das veröffentlichte Chart ändert (neue Version, anderes Chart, anderes Dogu),
muss das Mock entsprechend angepasst werden. Haltet Folgendes synchron:

- Die `Version` (und `Chart`, falls es verschoben wurde) in `nexus.json` /
  `nexus-not-found.json`.
- Die `spec.version` in den entsprechenden `Dogu`-CR(s).
- Die Configmap-Schlüssel im Ziel `mock-dcc-v3` im
  [`Makefile`](../../Makefile) (`--from-file=<version>=…`), die mit den
  Versionen übereinstimmen müssen, die die `Dogu`-CRs anfordern.

Im Fall von „helm-404“ richtet den Deskriptor auf eine Version aus, die **nicht**
veröffentlicht ist.