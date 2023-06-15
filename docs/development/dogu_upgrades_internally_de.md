# Dogu-Upgrades

Ein Dogu-Upgrade verläuft in folgenden Schritten:

1. Das Image von DoguV2 wird gepullt.
2. Das Pre-Upgrade-Skript von DoguV2 wird in DoguV1 kopiert und ausgeführt
3. DoguV1 wird heruntergefahren
4. DoguV2 wird hochgefahren und wartet zunächst mit eigentlichem Start
5. Das Post-Upgrade-Skript von DoguV2 wird ausgeführt
6. DoguV2 setzt seine Startroutine fort

## Pre-Upgrade

### Entscheidungsfindung
Im Gegensatz zum herkömmlichen CES ist es nicht so einfach möglich von einem Image Dateien in einen laufenden Container
zu kopieren und dort auszuführen. Ad Hoc ein Volume zu mounten würde einen Neustart des Containers verursachen.
Dies gilt zu verhindern, da die eigentliche Anwendung ebenfalls laufen muss. Bei z.B. Dogus wie EasyRedmine würde dies
unnötig Zeit in Anspruch nehmen.

Eine weitere Idee war, das Skript per cat zu extrahieren und als HEREDOC in den Container einzufügen.  
Wegen der Abhängigkeit zu chmod und Unklarheiten wie die das HEREDOC an die Kubernetes API übergeben 
werden soll, haben wir uns gegen diese Lösung entschieden.

Es wurde auch überlegt, statt des ExecPod einen dauernd laufenden Sidecar zu verwenden, diese Idee wurde allerdings 
aufgrund der damit einhergehenden Ressourcenverschwendung verworfen.

`kubectl cp` nutzt `tar` um Dateien und Verzeichnisse als Archiv zu verpacken und am Zielort zu entpacken. 
Eine Möglichkeit ist es analog vorzugehen und somit kein zusätzliches Volume zu benötigen.

### ExecPod
Das Pre-Upgrade Skript kommt aus dem neuen Container und wird auf den alten Container angewandt.  
Dafür startet der Dogu-Operator beim Upgrade einen ExecPod des neuen Dogus und kopiert das Skript mittels `tar` in das alte Dogu.  
ExecPods benutzen das Image der neuen Dogu-Version, werden allerdings mit Sleep Infinity gestartet.

### Ausführen des Pre-Upgrade-Skripts
Das Pre-Upgrade-Skript wird dann im alten Container vom ursprünglichen Pfad aus durch den Dogu-Operator ausgeführt.

## Post-Upgrade

### Ausführen des Post-Upgrade-Skripts
Der Dogu-Operator wartet bis alle Container des neuen Dogu Pods gestartet sind und startet dann das Post-Upgrade-Skript direkt im neuen Dogu-Pod.  
Ein ExecPod ist im Gegensatz zum Pre-Upgrade nicht nötig, da das benötigte Skript im neuen Image vorhanden ist.

## Probes während und nach dem Upgrade
Damit eventuell längere Startup Zeiten eines Dogus nach einem Upgrade abgefangen werden, wird nach einem Upgrade der
FailureThreshold der Startup Probe hochgesetzt.  
Nach dem erfolgreichen Upgrade wird diese Änderung wieder zurückgesetzt.
