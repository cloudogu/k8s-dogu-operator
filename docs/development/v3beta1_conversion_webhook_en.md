# The `v3beta1` conversion webhook

## Background: a second served API version

`k8s-dogu-lib/v3` defines two API versions for the Dogu CRD: `v2` and `v3beta1`. Both are **served**.
`v2` is the **storage version**.
Because two served versions with a non-trivial schema difference exist side by side, the Kubernetes API
server needs a **conversion webhook** for the Dogu CRD, regardless of which one is the storage version.
The operator implements the webhook with the kubebuilder framework.
For API server to Webhook communication, additional resources are required.

## Helm resources

### `cert-manager.yaml`

A self-signed `Issuer` (`<name>-selfsigned-issuer`) and a `Certificate` named literally
`k8s-dogu-operator-webhook-cert` — this exact name is a **contract with `k8s-dogu-lib`**, do not rename
it without checking the lib. `Certificate.spec.secretName` matches, and its DNS names are
`<name>-webhook.<namespace>.svc[.cluster.local]`.

This is why the operator has **cert-manager as a runtime dependency**: it issues and rotates the TLS
certificate the webhook server serves. See [install_cert_manager_en.md](install_cert_manager_en.md) for
how to install it in a cluster.

### `webhook-service.yaml`

A `Service` named `<name>-webhook` (again a contract with `k8s-dogu-lib`), port `443` →
`targetPort: webhook-server` (matching the container port name in `deployment.yaml`).

It sets `publishNotReadyAddresses: true` deliberately: the apiserver calls this Service to run the
conversion webhook, and on operator startup all Dogu resources immediately trigger a reconcile.
*If* the storage version were ever switched to `v3beta1`, every one of those startup reconciles would need the webhook
right away — but the webhook server takes ~5-10 seconds to become ready. Without `publishNotReadyAddresses`, there would
be no Service endpoints yet, those webhook calls would fail, the reconciles would fail, and the pod could get
stuck never reaching readiness (a self-inflicted deadlock). With `v2` as the storage version, most
startup reconciles don't hit the webhook at all, so the risk is currently latent rather than active —
but the setting is in place either way.

### `deny-all-network-policy.yaml`

Gated by `.Values.global.networkPolicies.enabled`. Denies all ingress to the operator pods except TCP
port `9443` (the webhook port). That port is deliberately left open to all sources because the
kube-apiserver calls the conversion webhook directly, and its source IP is unpredictable.

### `deployment.yaml`

The manager container exposes `containerPort: 9443, name: webhook-server` and mounts a `webhook-cert`
volume (a `secret` volume, `secretName: k8s-dogu-operator-webhook-cert`, `optional: false`) at
`/tmp/k8s-webhook-server/serving-certs` — controller-runtime's default webhook cert path.
