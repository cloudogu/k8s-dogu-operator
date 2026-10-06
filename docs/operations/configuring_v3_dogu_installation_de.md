# Konfiguration für die Installation von v3-Dogus

## Flux-Abhängigkeiten

v3-Dogus werden als Helm-Charts mithilfe von zwei Flux-Controllern installiert: source-controller und helm-controller.
Beide müssen zusammen mit ihren CRDs installiert werden, damit v3-Dogus installiert werden können. Dies kann
mithilfe von ecosystem-core erfolgen, indem dessen Wert für `flux.enabled` auf `true` gesetzt wird. 
Neben Installations- und Upgrade-Vorgängen wird dadurch auch die Drift-Erkennung für Dogu-Installationen aktiviert, 
die manuelle Änderungen am zugehörigen Cluster-Status automatisch rückgängig macht.

## Konfiguration

Der Dogu-Operator benötigt einige zusätzliche Konfigurationen, um die Artefakte für v3-Dogus zu finden und abzurufen:

Das Secret `dogu-registry-v3` muss die URL und die Anmeldedaten für die v3-Dogu-Registry enthalten
(siehe [Konfiguration der Dogu-V3-Registry](configuring_the_dogu_registry_de.md)).

Dazu gibt es einige Konfigurationsoptionen, die per Umgebungsvariablen gesetzt werden, um festzulegen, 
wie die Helm-Installation gehandhabt wird:
  - `DOGU_HELM_RETRY_INTERVAL` steuert, wie lange vor einem erneuten Versuch gewartet werden soll, 
    falls die Helm-Installation oder das Upgrade fehlschlägt.
  - `DOGU_HELM_RECONCILIATION_INTERVAL` definiert das Intervall, in dem auf Abweichungen vom von Helm angeforderten Zustand geprüft wird,
    die dann automatisch korrigiert werden.

Beide Intervalle werden als Zahl mit einer Zeiteinheit angegeben, z. B. „20s“ (für 20 Sekunden), „2m“ (für 2 Minuten)
oder „1h“ (für 1 Stunde). Kombinationen wie „1h20m“ (für 1 Stunde und 20 Minuten) sind ebenfalls möglich.
