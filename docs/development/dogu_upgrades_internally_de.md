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

### ExecPod
Upgrade Skripte kommen aus dem neuen Container und werden auf den alten Container angewandt.  
Dafür startet der Dogu-Operator beim Upgrade einen ExecPod des neuen Dogus und kopiert das Skript in das _reserved_ Volume.  
ExecPods benutzen das Image der neuen Dogu-Version, werden allerdings mit Sleep Infinity gestartet.

### _reserved_ Volume
Jedes Dogu erhält bei der Installation ein _reserved_ Volume mit einem Persistent Volume Claim von 10 MiB Größe.  
Der Name des Volumes und des Claims lautet `<dogu-name>-reserved`.  
Aufgrund der limitierten Größe des Volumes sind Pre-Upgrade-Skripte in ihrer Größe auch limitiert.

### Ausführen der Pre-Upgrade-Skripte
Aus dem _reserved_ Volume werden die Pre-Upgrade-Skripte dann im alten Container an den ursprünglichen Pfad kopiert.  
Dann werden diese durch den Dogu-Operator ausgeführt.

## Probes während und nach dem Upgrade
Damit eventuell längere Startup Zeiten eines Dogus nach einem Upgrade abgefangen werden, wird nach einem Upgrade der
FailureThreshold der Startup Probe hochgesetzt.  
Nach dem erfolgreichen Upgrade wird diese Änderung wieder zurückgesetzt.
