# Kontinuierliche Reconciliation

Dogus werden kontinuierlich reconciled, bis der gewünschte Zustand der Dogu-Ressource erreicht ist.

## Neustart bei Konfigurationsänderung

Dies betrifft Neustarts, wenn sich die Konfiguration des Dogus ändert (zu finden in der ConfigMap und dem Secret mit dem
Namen `<dogu-name>-config`). Änderungen in der globalen Konfiguration lösen einen Neustart aller Dogus aus (zu finden in
der ConfigMap namens `global-config`).

## Reconciliation pausieren

Wenn eine kontinuierliche Reconciliation und automatische Neustarts in gewissen Fällen nicht gewünscht sind, kann die
Reconciliation vorübergehend über das Flag `spec.pauseReconciliation` der Dogu-Ressource pausiert werden.

Dadurch werden **KEINE** Änderungen an der Dogu-Ressource angewendet werden (außer `spec.pauseReconciliation`).
Die Validierung wird weiterhin ausgeführt.

**WARNUNG**: Aktivieren Sie diese Option nur vorübergehend, z. B. zu Debugging-Zwecken oder wenn Sie das Dogu
aktualisieren und gleichzeitig die Konfiguration ändern möchten, ohne dazwischen einen Neustart durchzuführen.