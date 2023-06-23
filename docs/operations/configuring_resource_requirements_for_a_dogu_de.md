# Konfigurieren von Ressourcenanforderungen für Dogus

Dieses Dokument beschreibt, wie die Ressourcenanforderungen (Limits und Requests) für ein Dogu konfiguriert und angewendet werden können.

 - **Ressourcen-Requests:** Geben die von einem Dogu minimal benötigten Ressourcen (CPU-Kerne, Memory, Ephemeral-Storage) an, damit das Dogu funktionstüchtig ist.
                            Der Kubernetes-Scheduler sorgt dafür, dass das Dogu auf einem Node mit ausreichenden Ressourcen gestartet wird.
 - **Ressourcen-Limits:** Geben die maximal erlaubte Menge an Ressourcen an, die ein Dogu verwenden darf.
                          Wenn das Limit für die CPU-Kerne überschritten wird, drosselt die Container-Runtime die verfügbaren CPU-Ressourcen für den Pod.
                          Wenn das Memory-Limit oder der Ephemeral-Storage überschritten werden, wird der jeweilige Pod "evicted" und neu gestartet.

## Voraussetzungen

* Ein betriebsbereites Cloudogu MultiNode EcoSystem

## Limits & Requests

Die Ressourcenanforderungen können für jedes Dogu angewendet werden.
Generell gibt es drei verschiedene Limits bzw. Requests:

1. **CPU-Kerne**: Mehr Informationen gibt es auf der offiziellen Seite von
   Kubernetes: [Kubernetes-CPU](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-cpu)
2. **Memory**: Mehr Informationen gibt es auf der offiziellen Seite von
   Kubernetes: [Kubernetes-Memory](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory)
3. **Ephemeral-Storage**: Mehr Informationen gibt es auf der offiziellen Seite von
   Kubernetes: [Kubernetes-Ephemeral-Storage](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#local-ephemeral-storage)

## Konfigurieren von Ressourcenanforderungen

Die Limits und Requests werden generell im Etcd konfiguriert. **Hinweis:** Das Setzen einer Ressourcenanforderung führt nicht automatisch zu einem
Neustart des Dogus. Die Anwendung muss explizit stattfinden. Dies wird im nächsten Abschnitt beschrieben.

Generell können in jedem `config`-Bereich eines Dogus unter dem Abschnitt `container_config` folgende Einträge gesetzt werden:

**CPU-Kerne**

- Schlüssel für Request: `config/<DOGU_NAME>/container_config/cpu_core_request`
- Schlüssel für Limit: `config/<DOGU_NAME>/container_config/cpu_core_limit`
- Optional
- Beschreibung: Setzt die CPU-Ressourcenanforderung für jeden gestarteten Pod des Dogus.
- Format:
  siehe https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes

**Memory**

- Schlüssel für Request: `config/<DOGU_NAME>/container_config/memory_request`
- Schlüssel für Limit: `config/<DOGU_NAME>/container_config/memory_limit`
- Optional
- Beschreibung: Setzt die Memory-Ressourcenanforderung für jeden gestarteten Pod des Dogus.
- Format: Die konfigurierbaren Werte für die Schlüssel sind jeweils eine Zeichenkette der Form `<Zahlenwert><Einheit>` und beschreiben die maximal vom Dogu nutzbare Menge an Speicher.
  Zu beachten ist hier, dass zwischen dem numerischen Wert und der Einheit kein Leerzeichen stehen darf.
  Verfügbare Einheiten sind `b`, `k`, `m` und `g` (für byte, kibibyte, mebibyte und gibibyte). 
   **Hinweis:** Hier wird nicht das von Kubernetes verwendete Format benutzt!

**Ephemeral-Storage**

- Schlüssel für Request: `config/<DOGU_NAME>/container_config/storage_request`
- Schlüssel für Limit: `config/<DOGU_NAME>/container_config/storage_limit`
- Optional
- Beschreibung: Setzt die Ephemeral-Storage-Ressourcenanforderung für jeden gestarteten Pod des Dogus.
- Format: Die konfigurierbaren Werte für die Schlüssel sind jeweils eine Zeichenkette der Form `<Zahlenwert><Einheit>` und beschreiben die maximal vom Dogu nutzbare Menge an Speicher.
  Zu beachten ist hier, dass zwischen dem numerischen Wert und der Einheit kein Leerzeichen stehen darf.
  Verfügbare Einheiten sind `b`, `k`, `m` und `g` (für byte, kibibyte, mebibyte und gibibyte).
  **Hinweis:** Hier wird nicht das von Kubernetes verwendete Format benutzt!


## Konfigurierte Ressourcenanforderungen anwenden

Damit die Ressourcenanforderungen auch angewendet werden, muss der globale Etcd-Key: `config/_global/sync_resource_requirements`
erstellt/verändert/gelöscht werden. Jede Änderung an dem Schlüssel führt zum Start einem automatischen Update Prozess
für alle Dogus. In diesem Update Prozess werden für alle Dogus die Ressourcenanforderungen angewendet und die Dogus, wenn neue Ressourcenanforderungen
gesetzt wurden, neu gestartet. Unveränderte Dogus werden nicht neu gestartet. 
Generell kann der Update-Prozess mit folgendem Befehl gestartet werden:

```bash
etcdctl set /config/_global/sync_resource_requirements true
```
