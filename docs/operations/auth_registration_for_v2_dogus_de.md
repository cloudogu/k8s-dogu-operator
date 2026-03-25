# AuthRegistration für v2-Dogus

Dieses Feature steuert, wie der `k8s-dogu-operator` SSO-Registrierungen für v2-Dogus anlegt.

## Konfiguration

Die Funktion wird über die Umgebungsvariable `AUTH_REGISTRATION_ENABLED` gesteuert.

- `false` (Standard): Legacy-Verhalten über CAS-Service-Account-Erstellung, wenn das CAS-Dogu installiert ist
- `true`: wenn LOP-IdP und AuthRegistration-CRs verwendet werden sollen

Bei Helm-Installationen wird der Wert über `values.yaml` gesetzt:

```yaml
controllerManager:
  env:
    authRegistrationEnabled: true
```

## Verhalten bei `AUTH_REGISTRATION_ENABLED=true`

- Für v2-Dogus mit einem Legacy-CAS-Service-Account in der `dogu.json` erstellt/aktualisiert der Operator eine `AuthRegistration`-CR.
- Die Parameter werden aus den Legacy-Parametern gelesen (`account_type [logout_uri]`).
- Unterstützte Protokolle: `cas`, `oidc`, `oauth` (case-insensitive).
- Die durch die AuthRegistration aufgelösten Zugangsdaten werden aus dem `status.resolvedSecretRef`-Secret in die sensitive Dogu-Config übernommen:
  - Zielpfad: `/sa-<serviceAccountType>/<key>`
- Solange `resolvedSecretRef` fehlt oder das Secret noch keine Inhalte enthält, wird die Dogu-Reconciliation mit einer Meldung requeued.
- Die Legacy-CAS-Service-Account-Erstellung wird in diesem Modus übersprungen.

## Verhalten bei `AUTH_REGISTRATION_ENABLED=false`

- Der Operator nutzt unverändert die bisherige Legacy-Service-Account-Logik (z. B. CAS-Skriptpfad).
- Es werden keine AuthRegistration-CRs verwaltet.