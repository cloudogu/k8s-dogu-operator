# Leitfaden zur Entwicklung

## Lokale Entwicklung

1. Befolgen Sie die Installationsanweisungen von k8s-ecosystem
2. Bearbeiten Sie Ihre `/etc/hosts` und fügen Sie ein Mapping von localhost zu etcd hinzu
   - `127.0.0.1       localhost etcd etcd.ecosystem.svc.cluster.local`
3. port-forward zu etcd (Prozess blockiert und sollte in einem neuen Terminal ausgeführt werden)
    ```bash
    kubectl port-forward -n=ecosystem etcd-0 4001:2379
    ```
4. Führen Sie `make manifests` aus
5. Führen Sie `make install` aus
6. Öffnen Sie die Datei `.myenv.template` und folgen Sie den Anweisungen um eine 
   Umgebungsvariablendatei mit persönlichen Informationen anzulegen
7. Run `make run` to run the dogu operator locally
8. Führen Sie `make run` aus, um den dogu-Operator lokal auszuführen

## Makefile-Targets

Der Befehl `make help` gibt alle verfügbaren Targets und deren Beschreibungen in der Kommandozeile aus.

## Lokaler Image-Build

Um lokal das Image des `dogu-operator` zu bauen wird im Projektverzeichnis ein `.netrc`-File benötigt.

```
machine github.com
login <username>
password <token>
```

Der Token benötigt Berechtigungen, um private Repositorys zu lesen.

## Verwendung von benutzerdefinierten Dogu-Deskriptoren

Der `dogu-operator` ist in der Lage für ein Dogu eine benutzerdefinierte `dogu.json` bei der Installation zu verwenden.
Diese Datei muss in Form einer Configmap im selben Namespace liegen. Der Name der Configmap muss `<dogu>-descriptor`
lauten und die Nutzdaten müssen in der Data-Map unter dem Eintrag `dogu.json` verfügbar sein.
Es existiert ein Make-Target zur automatischen Erzeugung der Configmap - `make install-dogu-descriptor`.
Dabei ist zu beachten, dass der Dateipfad unter der Variable `CUSTOM_DOGU_DESCRIPTOR` exportiert werden muss.

Nach einer erfolgreichen Dogu-Installation, wird die Configmap gelöscht.

## Filtern der Reconcile-Funktion

Damit die Reconcile-Funktion nicht unnötig aufgerufen wird, wenn die Spezifikation eines Dogus sich nicht ändert,
wird der `dogu-operator` mit einem Update-Filter gestartet. Dieser Filter betrachtet das Feld `generation` der alten
und neuen Dogu-Ressource. Wird bei der Dogu-Ressource ein Feld der Spezifikation geändert inkrementiert die K8s-Api
`generation`. Bei einer Gleichheit des Feldes von dem alten und neuen Dogu wird das Update nicht betrachtet.
