# Dogu-Upgrades

Ein Dogu-Upgrade stellt auf den ersten Blick nicht mehr dar, als eine neue Dogu-Version in das Cloudogu EcoSystem
einspielen. Ein Dogu-Upgrade ist eine von mehreren Operationen, die `k8s-dogu-operator` unterstützt. Grundsätzlich ist
es nur möglich, Dogus mit einer höheren Version zu aktualisieren. Sonderfälle diskutiert der Abschnitt "
Upgrade-Sonderfälle"

Ein solches Upgrade lässt sich leicht erzeugen.

**Beispiel:**

Ein Dogu wurde bereits in einer älteren Version mit dieser Dogu-Resource mittels `kubectl apply` installiert:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: my-dogu
  labels:
    dogu.name: my-dogu
    app: ces
spec:
  name: official/my-dogu
  version: 1.2.3-4
```

Ein Upgrade des Dogus auf Version `1.2.3-5` ist denkbar einfach. Eine vergleichbare Resource mit neuerer Version
herstellen und wieder mit `kubectl apply ...` auf den Cluster anwenden:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: my-dogu
  labels:
    dogu.name: my-dogu
    app: ces
spec:
  name: official/my-dogu
  version: 1.2.3-5
```

## Pre-Upgrade-Skript

Für das Pre-Upgrade-Skript wird während des Upgrade-Prozesses ein Pod gestartet.
Dieser verwendet das aktualisierte Image des Dogus und kopiert nur das in der Dogu.json genannte Skript in den alten
Container.

Ein dafür vorgesehenes Volume wird bereits bei der Installation angelegt. Nachdem das Pre-Upgrade-Skript im alten
Container verfügbar gemacht wurde, wird dies ausgeführt während das Dogu läuft.

### Anforderungen an ein Pre-Upgrade-Skript

Dieser Abschnitt definiert leicht umzusetzende Anforderungen für Dogu-Entwickelnde, um die Ausführung von
Pre-Upgrade-Skripten so fehlerfrei und transparent wie möglich zu gestalten.

#### Parameter

Pre-Upgrade-Skripte müssen genau zwei Parameter entgegennehmen:

1. die alte Dogu-Version, die gerade läuft
2. die neue Dogu-Version, auf die das Upgrade angewendet werden soll

Anhand dieser Informationen können Pre-Upgrade-Skripte entscheidende Entscheidungen treffen. Dies kann u. a. sein:
- Verweigerung von Upgrades für zu große Versionssprünge
- angepasste Vorbereitungsmaßnahmen je vorgefundener Version

Beispielsweise könnte das Pre-Upgrade-Skript so aufgerufen werden:

```bash
/path/to/pre-upgrade.sh 1.2.3-4 1.2.3-5
```

Die Übergabe weiterer Parameter ist nicht vorgesehen.

#### Nutzung von absoluten Dateireferenzen

Wenn es um Dateiverarbeitung geht, dann müssen Pre-Upgrade-Skripte absolute Dateipfade verwenden, da nicht sichergestellt werden kann, dass ein Skript immer von seinem Ursprungsort aus aufgerufen wird.

#### Keine Nutzung anderer Dateien

Pre-Upgrade-Skripte werden vom Upgrade-Image hin zum Dogu-Container kopiert, um dort ausgeführt zu werden. Da in dem Dogu-Deskriptor `dogu.json` ausschließlich das Pre-Upgrade-Skript und nicht zusammengehörige Dateien genannt werden können, muss ein Pre-Upgrade-Skripte in seinem Funktionsumfang vollumfänglich aufgebaut sein.

Dies schließt insbesondere das Shell-Sourcing anderer Dateien aus, da hierbei häufig falsche Annahmen von Versionsständen zu Fehlern führen.

#### Ausführbarkeit

- Das SetUID-Bit kann für Pre-Upgrade-Skripte aktuell nicht verwendet werden, da dieses durch aufruf von `cp` verloren geht
- `/bin/cp` muss zwingend installiert sein
- Es wird davon ausgegangen, dass es sich beim Pre-Upgrade-Script um ein Shellskript und nicht um ein sonstiges
  Executable handelt (etwa eine Linux-Binärdatei)
   - Sollte dies nicht der Fall sein, so muss das Container-Image so aufgebaut sein, dass der Kopiervorgang mit dem
     jeweils aktuellen Container-Benutzer sowie die Ausführung des Executables möglich ist.
- Das Pre-Upgrade-Skript wird durch den aktuellen Container-User im alten Dogu ausgeführt

## Post-Upgrade Script

Unlike pre-upgrade scripts, post-upgrade scripts are subject to only minor constraints because the script is usually already in its execution location. The post-upgrade script is executed in the new dogu at the end of the upgrade process. The dogu is responsible for waiting for the post-upgrade script to finish. This is where the use of the dogu state has proven helpful:

```bash
# post-upgrade.sh
doguctl state "upgrading
# upgrade routines go here...
doguctl state "starting
```

```bash
# startup.sh
while [[ "$(doguctl state)" == "upgrading" ]]; do
  echo "Upgrade script is running. Waiting..."
  sleep 3
done
# regular start-up goes here
```

After that the upgrade is finished.

## Upgrade-Sonderfälle

### Downgrades

Downgrades von Dogus sind dann problematisch, wenn die neuer Dogu-Version die Datengrundlage der älteren Version durch
das Upgrade auf eine Weise modifiziert, dass die ältere Version mit den Daten nichts mehr anfangen kann. **Unter
Umständen wird das Dogu damit arbeitsunfähig**. Da dieses Verhalten sehr stark vom Werkzeughersteller abhängt, ist es im
allgemeinen nicht möglich, Dogus zu _downgraden_.

Daher verweigert der Dogu-Operator ein Upgrade einer Dogu-Resource auf eine niedrigere Version. Dieses Verhalten lässt
sich durch den Schalter `spec.upgradeConfig.forceUpgrade` mit einem Wert von True ausschalten.

**Achtung möglicher Datenschaden:***
Sie sollten vorher klären, dass das Dogu keinen Schaden durch das Downgrade nimmt.

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: cas
  labels:
    dogu.name: cas
    app: ces
spec:
  name: official/cas
  version: 6.5.5-3
  upgradeConfig:
    # für ein Downgrade von v6.5.5-4
    forceUpgrade: true
```

### Wechsel eines Dogu-Namespaces

Ein Dogu-Namespace-Wechsel wird durch eine Änderung der Dogu-Resource ermöglicht. Dies kann z. B. nötig sein, wenn ein
neues Dogu in einen anderen Namespace veröffentlicht wird.

Dieses Verhalten lässt sich durch den Schalter `spec.upgradeConfig.allowNamespaceSwitch` mit einem Wert von True
ausschalten.

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: cas
  labels:
    dogu.name: cas
    app: ces
spec:
  name: official/cas
  version: 6.5.5-4
  upgradeConfig:
    allowNamespaceSwitch: true
```
