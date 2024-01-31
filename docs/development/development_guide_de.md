# Leitfaden zur Entwicklung

## Lokale Entwicklung

1. Befolgen Sie die Installationsanweisungen von k8s-ecosystem
2. Bearbeiten Sie Ihre `/etc/hosts` und fügen Sie ein Mapping von localhost zu etcd hinzu
   - `127.0.0.1       localhost etcd etcd.ecosystem.svc.cluster.local`
3. Öffnen Sie die Datei `.env.template` und folgen Sie den Anweisungen um eine 
   Umgebungsvariablendatei mit persönlichen Informationen anzulegen
4. Erzeugen Sie einen etcd Port-Forward
   - `kubectl -n=ecosystem port-forward etcd-0 4001:2379`
5. Führen Sie `make run` aus, um den dogu-Operator lokal auszuführen
6. Löschen Sie eventuelle Dogu-Operator-Deployments im Cluster, um Parallelisierungsfehler auszuschließen
   - `kubectl delete deployment k8s-dogu-operator`

### Debugging mit IntelliJ

1. Folgen Sie die oben beschriebenen Schritte, mit Ausnahme von `make run`
2. Benutzen Sie den Abschnitt zu IntelliJ aus dem .env-template
3. Lassen Sie sich Ihre Umgebungsvariablen mit `make print-debug-info` ausgeben
4. Kopieren Sie sich das Ergebnis in Ihre intelliJ Startkonfiguration
5. Starten Sie die main.go im Debug-mode

## Makefile-Targets

Der Befehl `make help` gibt alle verfügbaren Targets und deren Beschreibungen in der Kommandozeile aus.

Der Token benötigt Berechtigungen, um private Repositorys zu lesen.

## Verwendung von benutzerdefinierten Dogu-Deskriptoren

Der `dogu-operator` ist in der Lage für ein Dogu eine benutzerdefinierte `dogu.json` bei der Installation zu verwenden.
Diese Datei muss in Form einer Configmap im selben Namespace liegen. Der Name der Configmap muss `<dogu>-descriptor`
lauten und die Nutzdaten müssen in der Data-Map unter dem Eintrag `dogu.json` verfügbar sein.
Es existiert ein Make-Target zur automatischen Erzeugung der Configmap - `make install-dogu-descriptor`.

Nach einer erfolgreichen Dogu-Installation wird die Configmap gelöscht.

## Filtern der Reconcile-Funktion

Damit die Reconcile-Funktion nicht unnötig aufgerufen wird, wenn die Spezifikation eines Dogus sich nicht ändert,
wird der `dogu-operator` mit einem Update-Filter gestartet. Dieser Filter betrachtet das Feld `generation` der alten
und neuen Dogu-Ressource. Wird bei der Dogu-Ressource ein Feld der Spezifikation geändert inkrementiert die K8s-Api
`generation`. Bei einer Gleichheit des Feldes von dem alten und neuen Dogu wird das Update nicht betrachtet.
