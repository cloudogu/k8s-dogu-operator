# Dev charts for v3 dogus

In dev mode, the operator installs a v3 dogu from an arbitrary OCI chart instead of the dogu registry (DCC).
This allows testing a locally built chart in the cluster without releasing it first.

> **Attention:** The dev mode is meant for development clusters only and must stay disabled in production clusters.
> With dev mode enabled, everyone who may change dogu CRs can bring arbitrary charts into the cluster.

## Enabling the dev mode

The dev mode is controlled by the environment variable `DOGU_V3_DEV_CHARTS_ENABLED` of the operator, default is `false`.
In the Helm chart of the operator, this corresponds to the value `controllerManager.env.doguV3DevChartsEnabled`:

```yaml
controllerManager:
  env:
    doguV3DevChartsEnabled: true
```

## Annotations on the dogu CR

| Annotation | Meaning |
|---|---|
| `k8s.cloudogu.com/dev-chart-url` | OCI URL of the chart without tag, e.g. `oci://k3d-registry-proxy.localhost:5000/official/dogu/v3/charts/nexus` |
| `k8s.cloudogu.com/dev-chart-insecure` | Optional; `"true"` sets `insecure` on the `OCIRepository` (plain HTTP or no certificate verification) |

If `k8s.cloudogu.com/dev-chart-url` is set and dev mode is enabled, the operator does not query the dogu registry but sets `spec.url` of the `OCIRepository` to the annotation.
The tag stays `spec.version` of the dogu CR; every new dev state therefore needs a new version.

If the operator uses a dev chart, it writes an event with the reason `DevChart` to the dogu CR when creating or changing the `OCIRepository`.

If dev mode is disabled, the operator ignores the annotations, uses the dogu registry as usual and writes a warning event with the reason `DevChartsDisabled` to the dogu CR.

## Back to the regular dogu

Remove the annotation `k8s.cloudogu.com/dev-chart-url` and set `spec.version` to a version from the dogu registry.
On the next reconcile, the operator uses the dogu registry again.

The operator only reacts to changes of the spec of the dogu CR.
Changing only the annotations therefore takes effect once `spec.version` changes as well.
