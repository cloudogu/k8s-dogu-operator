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
Dieser verwendet das aktualisierte Image des Dogus und kopiert nur das Skript in den alten Container.
Ein dafür vorgesehenes Volume wird bereits bei der Installation angelegt.

### Herausforderung: Differenz zwischen Dateisystemlayout und aktuellem User

Durch die Kopie des Pre-Upgrade-Skripts vom neuen in den alten Container ergibt eine Problematik, wenn die Datei aus
Rechtegründen nicht kopiert werden kann, etwa wenn man sich das folgende Dateisystem vorstellt:

```
ls -lha / 
drwxr-xr-x    1 root     root        4.0K Dec 13 10:47 .
-rwxrwxr-x    1 root     root         704 Oct 12 14:25 pre-upgrade.sh
...

ls -lha /tmp/dogu-reserved/
drwxrwsr-x    3 root     doguuser    1.0K Dec 13 10:48 .
-rwxr-xr-x    1 doguuser doguuser     704 Dec 13 10:48 pre-upgrade.sh
...
```

Zur Lösung wurden mehrere Wege bedacht. Die folgenden vier Wege wurden abgewogen und für zu problemhaft bewertet:

1. Die Upgrade-Skripte werden stets mit dem zuletzt angegebenen User und dessen Rechten ausgeführt. Kopieren von
   Root-Dateien mit spezifischen Usern kommt daher nicht in Frage.
2. Da es vom Skriptautor abhängt, ob relative oder absolute Pfade im Skript verwendet werden, lässt sich die Datei auch
   nicht an einen anderen Ort kopieren und dort ausführen.
3. Gleiches gilt für eine Ausführung vom ursprünglich zu startenden Skript, das relativ an generell einem anderen Ort,
   weil d
4. Ein dynamisches Einführen von Anweisungen im Upgradeskript wird auch verworfen, diese Lösung einerseits komplex und
   fehleranfällig ist. Wir können nicht beliebige Dateipfade auswerten und umschreiben.

Stattdessen wurde sich für die folgende Lösung entschieden:

Diese besteht darin, in das Verzeichnis zu wechseln, für das das Upgradeskript konzipiert wurde. Dann wird der Inhalt
des Skripts durch Shell-Piping direkt durch den gewählten Skriptinterpreter ausgeführt.

- Mit diesem Snippet lässt sich dieses Verhalten im alten Dogu-Container testen:
- `sh -c "cd (basename /preupgrade.sh) && sh -c < /tmp/dogu-reserved/pre-upgrade.sh"`
   - hierbei muss das zweite Vorkommnis des Shellinterpreters `sh` durch einen im Skript definierten ausgetauscht
     werden, um eine maximale Kompatibilität von Skript und Interpreter zu gewährleisten.

### Einschränkungen

Durch das beschriebene Verhalten gelten damit die folgenden Einschränkungen für Pre-Upgrade-Skripte:

- Das SetUID-Bit kann für Pre-Upgrade-Skripte aktuell nicht verwendet werden, da dieses nicht durch `cp` verloren geht
- `/bin/cp` muss zwingend installiert sein
- `/bin/grep` muss in dem Fall installiert sein, wenn das Pre-Upgrade-Skript oder dessen Verzeichnis einen anderen
  Unix-User aufweisen, als in dem laufenden Dogu vorhanden
- Es wird davon ausgegangen, dass es sich beim Pre-Upgrade-Script um ein Shellskript und nicht um ein sonstiges
  Executable handelt (etwa eine Linux-Binärdatei)
   - Sollte dies nicht der Fall sein, so muss das Container-Image so aufgebaut sein, dass der Kopiervorgang mit dem
     jeweils aktuellen Container-Benutzer sowie die Ausführung des Executables möglich ist.

## Post-Upgrade-Skript

Das Post-Upgrade-Skript wird am Ende des Upgrade-Prozesses im neuen Dogu ausgeführt.  
Danach ist das Upgrade abgeschlossen.

## Upgrade-Sonderfälle

### Downgrades

To Do DG

### Wechsel eines Dogu-Namespaces

To Do DNW