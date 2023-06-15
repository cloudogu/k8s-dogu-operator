# Entwicklungsdiskussionen im Bereich Dogu-Upgrades

In diesem Dokument werden Entwicklungsentscheidungen diskutiert, die sich auf Upgrades von Dogus beziehen.

## Pre-Upgrade-Skripte

### Herausforderung: Differenz zwischen Dateisystemlayout und aktuellem User

Durch die Kopie des Pre-Upgrade-Skripts vom neuen in den alten Container ergibt eine Problematik, wenn die Datei aus
Rechtegründen nicht kopiert werden kann, etwa wenn man sich das folgende Dateisystem vorstellt:

```
ls -lha / 
drwxr-xr-x    1 root     root        4.0K Dec 13 10:47 .
-rwxrwxr-x    1 root     root         704 Oct 12 14:25 pre-upgrade.sh
...

ls -lha /tmp/dogu-reserved/
drwxrwxr-x    3 root     doguuser    1.0K Dec 13 10:48 .
-rwxr-xr-x    1 doguuser doguuser     704 Dec 13 10:48 pre-upgrade.sh
...
```

Zur Lösung wurden mehrere Wege bedacht. Die folgenden Wege wurden gegeneinander abgewogen:

1. Die Upgrade-Skripte werden stets mit dem zuletzt angegebenen User und dessen Rechten ausgeführt. Kopieren von
   Root-Dateien mit spezifischen Usern scheiter daher in der Regel.
   - Beispiel: `cp /tmp/dogu-reserved/pre-upgrade.sh / && /pre-upgrade.sh "${versionAlt}" "${versionNeu}"`
2. Da es vom Skriptautor abhängt, ob relative oder absolute Pfade im Skript verwendet werden, lässt sich die Datei auch
   nicht an einen anderen Ort kopieren und dort ausführen, ohne Fehler zu riskieren.
   - Beispiel: `cd /tmp/dogu-reserved && ./pre-upgrade.sh "${versionAlt}" "${versionNeu}"`
3. Gleiches gilt für eine Ausführung vom Arbeitsverzeichnis des ursprünglich zu startenden Skript
   - Beispiel: `cd / && /tmp/dogu-reserved/pre-upgrade.sh ${versionAlt}" "${versionNeu}"`
4. Ein dynamisches Einführen von Anweisungen im Upgradeskript wird auch verworfen, diese Lösung einerseits komplex und
   fehleranfällig ist. Es ist nicht ohne weiteres möglich, beliebige Dateipfade auszuwerten und umzuschreiben.
   - Beispiel: `sed -i 's|/|/tmp/dogu-reserved|g' /tmp/dogu-reserved/pre-upgrade.sh && /tmp/dogu-reserved/pre-upgrade.sh ${versionAlt}" "${versionNeu}"`
5. Skripts in eine Shell-Pipe lesen und den Stream durch einen geeigneten Interpreter ausführen lassen
   - Beispiel: `cd $(dirname /pre-upgrade.sh) && (cat /tmp/dogu-reserved/pre-upgrade.sh | /bin/bash -s ${versionAlt}" "${versionNeu}")`

Schlussendlich haben alle Lösungen sowohl Vorteile als auch Nachteile. Die geringste Komplexität bieten jedoch die Lösungen 2. und 3., die sich inhaltlich nur durch das Arbeitsverzeichnis unterscheiden. Hierbei wird die Komplexität durch Konventionen, wie [Pre-Upgrade-Skripte entwickelt](../operations/dogu_upgrades_de.md) werden müssen, eingetauscht.
