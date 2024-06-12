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
Container. Dieses wird dann im alten Dogu während der Laufzeit ausgeführt. Dies geschieht vom gleichen Pfad, in dem das Skript im neuen Dogu lag.

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

Wenn es um Dateiverarbeitung geht, dann müssen Pre-Upgrade-Skripte absolute Dateipfade verwenden,
da nicht sichergestellt werden kann, dass ein Skript immer von seinem Ursprungsort aus aufgerufen wird.

#### Keine Nutzung anderer Dateien

Pre-Upgrade-Skripte werden vom Upgrade-Image hin zum Dogu-Container kopiert, um dort ausgeführt zu werden.
Da in dem Dogu-Deskriptor `dogu.json` ausschließlich das Pre-Upgrade-Skript und nicht zusammengehörige Dateien genannt werden können,
muss ein Pre-Upgrade-Skript in seinem Funktionsumfang vollumfänglich aufgebaut sein.

Dies schließt insbesondere das Shell-Sourcing anderer Dateien aus, da hierbei häufig falsche Annahmen von Versionsständen zu Fehlern führen.

#### Ausführbarkeit

- Das SetUID-Bit kann für Pre-Upgrade-Skripte aktuell nicht verwendet werden, da dieses beim Kopieren von Pod zu Pod (mittels `tar`) verloren geht
- `/bin/tar` muss zwingend installiert sein
- Es wird davon ausgegangen, dass es sich beim Pre-Upgrade-Script um ein Shellskript und nicht um ein sonstiges
  Executable handelt (etwa eine Linux-Binärdatei)
   - Sollte dies nicht der Fall sein, so muss das Container-Image so aufgebaut sein, dass der Kopiervorgang mit dem
     jeweils aktuellen Container-Benutzer sowie die Ausführung des Executables möglich ist.
- Das Pre-Upgrade-Skript wird durch den aktuellen Container-User im alten Dogu ausgeführt

#### Limitierungen

Die Größe des Pre-Upgrade-Skriptes ist lediglich durch den Arbeitsspeicher limitiert.

## Post-Upgrade Skript

Im Gegensatz zum Pre-Upgrade-Skript unterliegt das Post-Upgrade-Skript nur geringen Einschränkungen, da sich das Skript in der Regel bereits an seinem Ausführungsort befindet.
Das Post-Upgrade-Skript wird am Ende des Upgrade-Prozesses im neuen Dogu ausgeführt.
Das Dogu ist dafür verantwortlich, auf die Beendigung des Post-Upgrade-Skripts zu warten.
Hier hat sich die Verwendung der lokalen Dogu-Config als hilfreich erwiesen:

```bash
# post-upgrade.sh
doguctl config "local_state" "upgrading"
# upgrade routines go here...
doguctl state "local_state" "starting"
```

```bash
# startup.sh
while [[ "$(doguctl config "local_state" -d "empty")" == "upgrading" ]]; do
  echo "Upgrade script is running. Waiting..."
  sleep 3
done
# regular start-up goes here
```

Danach ist das Upgrade beendet.

## Upgrade-Sonderfälle

### Downgrades

Downgrades von Dogus sind dann problematisch, wenn die neuere Dogu-Version die Datengrundlage der älteren Version durch
das Upgrade auf eine Weise modifiziert, dass die ältere Version mit den Daten nichts mehr anfangen kann. **Unter
Umständen wird das Dogu damit arbeitsunfähig**. Da dieses Verhalten sehr stark vom Werkzeughersteller abhängt, ist es im
Allgemeinen nicht möglich, Dogus zu _downgraden_.

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
