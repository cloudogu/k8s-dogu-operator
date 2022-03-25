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
6. Exportieren Sie Ihre CES-Instanz-Zugangsdaten, damit der Operator sie verwenden kann
    - `export CES_REGISTRY_USER=instanceId && export CES_REGISTRY_PASS='instanceSecret'`
7. Exportieren Sie Ihren CES-Instanz-Namespace
   - `export NAMESPACE=ecosystem`
8. Führen Sie `make run` aus, um den dogu-Operator lokal auszuführen

## Makefile-Targets

Der Befehl `make help` gibt alle verfügbaren Targets und deren Beschreibungen in der Kommandozeile aus.

## Verwendung von benutzerdefinierten Dogu-Deskriptoren

Der `dogu-operator` ist in der Lage für ein Dogu eine benutzerdefinierte `dogu.json` bei der Installation zu verwenden.
Diese Datei muss in Form einer Configmap im selben Namespace liegen. Der Name der Configmap muss `<dogu>-descriptor`
lauten und die Nutzdaten müssen in der Data-Map unter dem Eintrag `dogu.json` verfügbar sein.
Es existiert ein Make-Target zur automatischen Erzeugung der Configmap - `make generate-dogu-descriptor`.
Dabei ist zu beachten, dass der Dateipfad unter der Variable `CUSTOM_DOGU_DESCRIPTOR` exportiert werden muss.

Nach einer Dogu-Installation wird in der ConfigMap das Dogu als Owner eingetragen. Wenn man das Dogu anschließend
deinstalliert, wird auch die Configmap aus dedm Cluster entfernt.