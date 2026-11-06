# Dev charts for v3 dogus

In dev mode, the operator installs a v3 dogu from an arbitrary OCI chart instead of the dogu registry (DCC).
This allows testing a locally built chart in the cluster without releasing it first.

> **Attention:** The dev mode is meant for development clusters only and must stay disabled in production clusters.
> With dev mode enabled, everyone who may change dogu CRs can bring arbitrary charts into the cluster.

## `cldk dogu dev-push`

Usually, the annotations do not have to be set by hand.
The recommended way is [`cldk dogu dev-push`](https://github.com/cloudogu/cldk), run from the dogu repository:

```shell
cldk dogu dev-push
```

The command builds the images, pushes them and the chart under a new dev version (`<version>-dev.<timestamp>`) to a dev registry and creates or updates the dogu CR.
It sets `spec.version`, `spec.doguApiVersion: v3` and the annotations described below.
Other fields such as `values` or `mappedValues` are kept.
As every push creates a new version, the operator picks up the new state right away.

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
Changing just the annotations therefore only takes effect once `spec.version` changes as well.
