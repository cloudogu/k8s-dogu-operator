# Dev-Charts für v3-Dogus

Im Dev-Modus installiert der Operator ein v3-Dogu aus einem beliebigen OCI-Chart statt aus der Dogu-Registry (DCC).
So lässt sich ein lokal gebautes Chart im Cluster testen, ohne es vorher zu releasen.

> **Achtung:** Der Dev-Modus ist nur für Entwicklungscluster gedacht und muss in Produktivclustern ausgeschaltet bleiben.
> Mit eingeschaltetem Dev-Modus kann jeder, der Dogu-CRs ändern darf, beliebige Charts in den Cluster bringen.

## `cldk dogu dev-push`

Die Annotations müssen in der Regel nicht von Hand gesetzt werden. 
Der empfohlene Weg ist [`cldk dogu dev-push`](https://github.com/cloudogu/cldk) aus dem Dogu-Repository heraus:

```shell
cldk dogu dev-push
```

Der Befehl baut die Images, pusht sie und das Chart unter einer neuen Dev-Version (`<version>-dev.<Zeitstempel>`) in eine Dev-Registry und legt die Dogu-CR an bzw. aktualisiert sie.
Dabei setzt er `spec.version`, `spec.doguApiVersion: v3` und die unten beschriebenen Annotations.
Andere Felder wie `values` oder `mappedValues` bleiben erhalten.
Da jeder Push eine neue Version erzeugt, übernimmt der Operator den neuen Stand sofort.

## Dev-Modus einschalten

Der Dev-Modus wird über die Umgebungsvariable `DOGU_V3_DEV_CHARTS_ENABLED` des Operators geschaltet, Default ist `false`.
Im Helm-Chart des Operators entspricht das dem Value `controllerManager.env.doguV3DevChartsEnabled`:

```yaml
controllerManager:
  env:
    doguV3DevChartsEnabled: true
```

## Annotations an der Dogu-CR

| Annotation | Bedeutung |
|---|---|
| `k8s.cloudogu.com/dev-chart-url` | OCI-URL des Charts ohne Tag, z.B. `oci://k3d-registry-proxy.localhost:5000/official/dogu/v3/charts/nexus` |
| `k8s.cloudogu.com/dev-chart-insecure` | Optional; `"true"` setzt `insecure` am `OCIRepository` (Plain-HTTP bzw. ohne Zertifikatsprüfung) |

Ist `k8s.cloudogu.com/dev-chart-url` gesetzt und der Dev-Modus an, fragt der Operator nicht die Dogu-Registry, sondern setzt `spec.url` des `OCIRepository` auf die Annotation.
Der Tag bleibt `spec.version` der Dogu-CR; jeder neue Dev-Stand braucht also eine neue Version.

Nutzt der Operator ein Dev-Chart, schreibt er beim Anlegen oder Ändern des `OCIRepository` ein Event mit dem Grund `DevChart` an die Dogu-CR.

Ist der Dev-Modus aus, ignoriert der Operator die Annotations, nutzt die Dogu-Registry wie gewohnt und schreibt ein Warning-Event mit dem Grund `DevChartsDisabled` an die Dogu-CR.

## Zurück zum regulären Dogu

Annotation `k8s.cloudogu.com/dev-chart-url` entfernen und `spec.version` auf eine Version aus der Dogu-Registry setzen.
Beim nächsten Reconcile nutzt der Operator wieder die Dogu-Registry.

Der Operator reagiert nur auf Änderungen an der Spec der Dogu-CR.
Eine Änderung nur an den Annotations wirkt deshalb erst, wenn sich auch `spec.version` ändert.
