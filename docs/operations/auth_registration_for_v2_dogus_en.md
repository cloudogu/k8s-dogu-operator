# AuthRegistration for v2 Dogus

This feature controls how the `k8s-dogu-operator` creates SSO registrations for v2 Dogus.

## Configuration

The feature is controlled by the `AUTH_REGISTRATION_ENABLED` environment variable.

- `false` (default): legacy CAS service-account registration, when using the CAS-Dogu
- `true`: when using LOP-IdP and AuthRegistration CRs

For Helm deployments, set it in `values.yaml`:

```yaml
controllerManager:
  env:
    authRegistrationEnabled: true
```

## Behavior when `AUTH_REGISTRATION_ENABLED=true`

- For v2 Dogus with a legacy CAS service account in `dogu.json`, the operator creates/updates an `AuthRegistration` CR.
- Parameters are read from legacy params (`account_type [logout_uri]`).
- Supported protocols: `cas`, `oidc`, `oauth` (case-insensitive).
- Credentials resolved by AuthRegistration are read from the `status.resolvedSecretRef` secret and synced to sensitive Dogu config:
  - target path: `/sa-<serviceAccountType>/<key>`
- If `resolvedSecretRef` is missing or the secret has no credential content yet, Dogu reconciliation is requeued with a message.
- Legacy CAS service-account creation is skipped in this mode.

## Behavior when `AUTH_REGISTRATION_ENABLED=false`

- The operator keeps using the existing legacy service-account flow (for example CAS script-based registration).
- No AuthRegistration CRs are managed.
