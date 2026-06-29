# WarpMenuEntry für v2 Dogu

Dieses Feature steuert, wie der `k8s-dogu-operator` `WarpMenuEntry`-Custom-Resources für v2-Dogus erstellt.


## Verhalten 

Der Operator erstellt/aktualisiert/löscht im Rahmen des regulären Reconcile-Ablaufs ein `WarpMenuEntry`-CR mit demselben Namen wie das Dogu(V2).

Damit ein Dogu im Warp-Menü angezeigt wird, muss ein `WarpMenuEntry`-CR vorhanden sein.

Früher erstellte der Operator „k8s-ces-asserts (warpmenu)“ das Warp-Menü auf der Grundlage der Datei „dogu.json“.
Der Operator „k8s-ces-asserts (warpmenu)“ erstellt das Warp-Menü nicht mehr auf Basis der Datei „dogu.json“, sondern ausschließlich auf Basis der „WarpMenuEntry“-CRs. 
Daher müssen wir im Rahmen des regulären Reconcile-Ablaufs des „k8s-dogu-operator“ einen „WarpMenuEntry“-CR mit demselben Namen wie das Dogu erstellen, aktualisieren oder löschen.

Zu beachten ist dabei, dass der `WarpMenuEntry`-CR nur erstellt wird, wenn in der Datei „dogu.json“ ein `warp`-Tag vorhanden ist,
andernfalls stellt der Reconcile-Ablauf sicher, dass kein WarpMenuEntry mit dem Namen des Dogu vorhanden ist.
Wenn ein `WarpMenuEntry`-CR existiert, sich jedoch der Anzeigename, die Kategorie oder der Pfad geändert hat, wird der `WarpMenuEntry`-CR während des Reconcile aktualisiert.

## Format des zu erstellenden WarpMenuEntry

### Für Dogus, die ein Warp-Tag haben

Wenn das Dogu über eine `warp`-Tag verfügt (z. B. „Jenkins-Dogu“), sollte ein entsprechender `WarpMenuEntry`-CR erstellt werden.

 dogu.json:
```json
{
  "Name": "official/jenkins",
  "DisplayName": "Jenkins",
  "Category": "Development Apps",
  "Tags": [
    "warp"
  ],
```

Daraus ergibt sich folgender `WarpMenuEntry`-CR:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: WarpMenuEntry
metadata:
  name: jenkins
  labels:
    app: ces
    app.kubernetes.io/name: k8s-warp-menu-entry-crd
spec:
  category: Development Apps
  disabled: false
  displayName:
    de: Jenkins
    en: Jenkins
  path: /jenkins
```

### Für Dogus ohne Warp-Tag

Wenn das Dogu kein Warp-Tag hat (z. B. ein LDAP-Dogu), wird kein `WarpMenuEntry`-CR erstellt:

dogu.json
```json
{
  "Name": "official/ldap",
  "DisplayName": "OpenLDAP",
  "Category": "Base",
  "Tags": [
    "authentication",
    "ldap",
    "users",
    "groups"
  ],
```
## Konfiguration und Codeablauf

### Konfiguration
Der k8s-dogu-Operator muss mehrere Ressourcen beachten und wurde daher in mehrere Schritte unterteilt.
Dieses Diagramm der Pakete/Dateien fasst die Liste aller erforderlichen Konfigurationen und Codeänderungen zusammen,
die für den Warp-Menüeintrag konfiguriert bzw. erstellt werden müssen:

![Konfiguration](../development/figures/configure_warp_menu_reconcile.png)

### Ablauf
Dies ist eine grafische Darstellung des Reconcile-Ablaufs für das Warp-Menü.
Der hervorgehobene Teil ist der, der den `WarpMenuEntry`-CR abgleicht.
Die Datei „main.go“ konfiguriert weitere Schritte während eines Dogu-Reconcile-Ablaufs.

![Reconcile-Ablauf](../development/figures/warp_menu_reconcile_flow.png)

