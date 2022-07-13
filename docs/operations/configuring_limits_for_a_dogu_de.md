# Konfigurieren von physikalischen Limits für Dogus

Dieses Dokument beschreibt, wie die physikalischen Limits für ein Dogu konfiguriert und angewendet werden können.

## Voraussetzungen

* Ein betriebsbereites Cloudogu MultiNode EcoSystem

## Physikalische Limits

Die physikalischen Limits können für jedes Dogu angewendet werden und beschränken den Pod des Dogus auf festgelegte
Limits.
Generell gibt es drei verschiedene Limits:

1. **CPU-Limit**: Mehr Informationen gibt es auf der offiziellen Seite von
   Kubernetes: [Kubernetes-CPU](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-cpu)
1. **Memory-Limit**: Mehr Informationen gibt es auf der offiziellen Seite von
   Kubernetes: [Kubernetes-Memory](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory)
1. **Ephemeral-Limit**: Mehr Informationen gibt es auf der offiziellen Seite von
   Kubernetes: [Kubernetes-Ephemeral-Storage](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#local-ephemeral-storage)

## Konfigurieren von Limits

Die Limits werden generell im Etcd konfiguriert. **Hinweis:** Das Setzen eines Limits führt nicht automatisch zu einem
Neustart des Dogus. Die Anwendung von Limits muss explizit stattfinden. Dies wird im nächsten Abschnitt beschrieben.

Generell können in jedem `config`-Bereich eines Dogus unter dem Abschnitt `pod_limit` folgende Einträge gesetzt werden:

**CPU-Limit**

- Schlüssel: `config/<DOGU_NAME>/pod_limit/cpu`
- Optional
- Beschreibung: Setzt das CPU-Limit für jeden gestarteten Pod des Dogus.
- Format:
  siehe https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes

**Memory-Limit**

- Schlüssel: `config/<DOGU_NAME>/pod_limit/memory`
- Optional
- Beschreibung: Setzt das Memory-Limit für jeden gestarteten Pod des Dogus.
- Format:
  siehe https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes

**Ephemeral-Storage-Limit**

- Schlüssel: `config/<DOGU_NAME>/pod_limit/ephemeral_storage`
- Optional
- Beschreibung: Setzt das Ephemeral-Storage-Limit für jeden gestarteten Pod des Dogus.
- Format:
  siehe https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes

## Konfigurierte Limits anwenden

Damit die Limits auch angewendet werden, muss der globale Etcd-Key: `config/_global/trigger-container-limit-sync`
erstellt/verändert/gelöscht werden. Jede Änderung an dem Schlüssel führt zum Start einem automatischen Update Prozess
für alle Dogus. In diesem Update Prozess werden für alle Dogus die Limits angewendet und die Dogus, wenn neue Limits
gesetzt wurden, neu gestartet. Unveränderte Dogus werden nicht neu gestartet. Generell kann der Update-Prozess mit dem 
Befehl:

```bash
etcdctl set /config/_global/trigger-container-limit-sync true
```

gestartet werden.