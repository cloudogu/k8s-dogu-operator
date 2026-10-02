# Mocking the DCC for Dogu V3 development

When developing the Dogu V3 code paths, the operator needs to talk to a Dogu
Registry that serves V3 dogu descriptors, and to an OCI registry that serves the dogus' Helm charts. 
This mock replaces the DCC's dogu registry with a tiny nginx pod so you can exercise the V3 fetch/render paths
(and their error cases) against a fully local, scripted fixture.

The OCI/Helm registry is **not** mocked — the operator still pulls the actual
Helm chart from `registry.cloudogu.com`. A chart for the referenced version
already exists there (see [Keeping the mock in sync](#keeping-the-mock-in-sync)).

All files live under [`dev/mock-dcc-v3/`](../../dev/mock-dcc-v3) and the entry
points are two Makefile targets: `mock-dcc-v3` and `mock-dcc-v3-clean`.

## Prerequisites

This setup has been tested with:

- A local **k3d** cluster.
- The **k8s-dogu-operator** running in the cluster with the V3 feature flag enabled: `controllerManager.env.doguV3Enabled: true` [`k8s/helm/values.yaml`](../../k8s/helm/values.yaml).
- **ecosystem-core** installed with `flux.enabled=true`. V3 dogus are installed as Helm charts via flux, so flux must be present in the cluster.

The targets operate in the namespace given by `NAMESPACE` in your `.env` (`ecosystem` by default).

## How it works

The operator discovers the dogu registry endpoint from the `dogu-registry-v3`
secret. The mock simply repoints that endpoint to an in-cluster nginx pod
(`test-dccv3`) that serves static descriptor files. Each descriptor tells the
operator which Helm chart (`Chart`) and which chart version (`Version`) to pull
from the real OCI registry.

The nginx pod mounts the `test-v3-dogus` configmap at
`/usr/share/nginx/html/api/v3/dogus/internal/nexus`, so the configmap keys
become the version path segments nginx serves.

## What gets deployed

Files under [`dev/mock-dcc-v3/`](../../dev/mock-dcc-v3):

| File | Purpose |
| --- | --- |
| `nexus.json` | Descriptor for the **success** version (valid chart + published version). |
| `nexus-not-found.json` | Descriptor for the **helm-404** version (valid chart, version not published). |
| `dcc.yaml` | The mock itself: nginx `Pod` + `Service` + `NetworkPolicy` (`test-dccv3`). |
| `dogu-success.yaml` | `Dogu` CR for the success scenario. |
| `dogu-404-helm-registry.yaml` | `Dogu` CR for the helm-404 scenario. |
| `dogu-404-dogu-registry.yaml` | `Dogu` CR for the dogu-registry-404 scenario. |

The descriptor files are published into the cluster as the `test-v3-dogus`
configmap, keyed by version. The `Dogu` CRs are **not** applied by the targets —
you apply the one you want to test by hand (see [Usage](#usage)).

## Usage

Set up the mock (idempotent — safe to re-run after editing descriptors):

```bash
make mock-dcc-v3
```

Then apply **one** scenario at a time:

```bash
kubectl -n ecosystem apply -f dev/mock-dcc-v3/dogu-success.yaml
kubectl -n ecosystem apply -f dev/mock-dcc-v3/dogu-404-helm-registry.yaml
kubectl -n ecosystem apply -f dev/mock-dcc-v3/dogu-404-dogu-registry.yaml
```

Watch the operator logs / the `Dogu` resource status to observe the outcome.

Tear everything down and restore the registry:

```bash
make mock-dcc-v3-clean
```

This deletes the nginx pod/service/networkpolicy and the configmap, then restores
the `dogu-registry-v3` endpoint from `.endpoint.bak` (or deletes the secret
entirely if the backup records `__ABSENT__`), and finally removes the backup
file. If no backup is found it warns and leaves the secret untouched.

## Keeping the mock in sync

The success scenario depends on a **real** Helm chart being published in the OCI
registry. A chart for the referenced version
(`…/charts/nexus/dogu-v3-develop:3.86.2-6`) already exists, so the success case
works out of the box.

When the published chart changes (new version, different chart, different dogu),
the mock must be adjusted to match. Keep the following in sync:

- The `Version` (and `Chart`, if it moved) in `nexus.json` /
  `nexus-not-found.json`.
- The `spec.version` in the corresponding `Dogu` CR(s).
- The configmap keys in the `mock-dcc-v3` target in the
  [`Makefile`](../../Makefile) (`--from-file=<version>=…`), which must equal the
  versions the `Dogu` CRs request.

For the helm-404 case, point the descriptor at a version that is **not**
published.
