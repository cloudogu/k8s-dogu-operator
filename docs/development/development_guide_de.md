# Leitfaden zur Entwicklung

## Lokale Entwicklung

1. Befolgen Sie die Installationsanweisungen von k8s-ecosystem
2. Bearbeiten Sie Ihre `/etc/hosts` und fügen Sie ein Mapping von localhost zu etcd hinzu
    - 127.0.0.1 localhost etcd etcd.ecosystem.svc.cluster.local".
3. port-forward zu etcd
    ```bash
    kubectl port-forward -n=oekosystem etcd-0 4001:2379
    ```
4. Führen Sie `make manifests` aus
5. Führen Sie `make install` aus
6. Exportieren Sie Ihre CES-Instanz-Zugangsdaten, damit der Operator sie verwenden kann
    - export CES_REGISTRY_USER=instanceId && export CES_REGISTRY_PASS='instanceSecret'`
7. Führen Sie `make run` aus, um den dogu-Operator lokal auszuführen

Dieses Dokument enthält Informationen über alle für diesen Controller verwendeten make-Targets.

## Zielübersicht (make)

Der Befehl `make help` gibt alle verfügbaren Targets und deren Beschreibungen in der Kommandozeile aus.