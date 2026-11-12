# Installing v3 dogus

Like a v2 dogu, a v3 dogu is installed via a dogu resource.
The difference lies in the API version of the resource and in the field `spec.doguApiVersion`, which specifies that the dogu is installed as a v3 dogu.
Installing v3 dogus requires the following prerequisites in the cluster.

The examples assume the namespace `ecosystem` and use the Nexus dogu.

## Prerequisites

- **Flux** is installed, via `ecosystem-core` with `flux.enabled: true`.
  The dogu operator uses Flux to roll out v3 dogus.
- **v3 dogus are enabled in the dogu operator**, see [Enabling v3 dogus in the operator](#enabling-v3-dogus-in-the-operator).
- **Secret `dogu-registry-v3`** contains the endpoint and credentials of the dogu registry v3, see [Secrets](#secrets).
- **Secret `ces-container-registries`** contains an entry for the registry the charts are pulled from (`registry.cloudogu.com` for official dogus).

### Enabling v3 dogus in the operator

v3 dogus are only processed if the operator's Helm value `controllerManager.env.doguV3Enabled` is `true`.
If it is not set, the operator ignores v3 dogus.
In the CES, it is set via the component resource of the operator:

```bash
kubectl -n ecosystem patch component k8s-dogu-operator --type=merge -p '
spec:
  valuesYamlOverwrite: |
    controllerManager:
      env:
        doguV3Enabled: true
'
```

> [!WARNING]
> The patch replaces the entire previous content of `valuesYamlOverwrite`.
> Existing overrides must therefore be included in the patch.

### Secrets

v3 dogus are fetched from the dogu registry v3.
For this, the operator needs its own credentials, separate from those for v2 dogus.

| Secret                     | Content                                                                              |
|----------------------------|--------------------------------------------------------------------------------------|
| `dogu-registry-v3`         | `endpoint`, `username`, `password`; optional `urlschema`, `insecureSkipVerify`        |
| `ces-container-registries` | Credentials of the container registry; created by the CES setup                      |
| `dogu-v3-registry-cert`    | optional: custom CA certificate (`dogu-v3-registry-cert.pem`) for the dogu registry v3 |

The `endpoint` of `dogu-registry-v3` contains the full path of the API, e.g. `https://dogu.cloudogu.com/api/v3/dogus`.
How to create the secret is described in [Configuring the dogu registry](../configuring_the_dogu_registry_en.md).

`ces-container-registries` is already used by v2 dogus.
For v3 dogus, it must additionally grant access to the registry that hosts the charts of the dogus.

> [!IMPORTANT]
> The operator reads `dogu-registry-v3` only at startup.
> After a change it must be restarted:
> `kubectl -n ecosystem rollout restart deployment k8s-dogu-operator-controller-manager`

### Further operator settings

The following Helm values are optional.
They define how often the state of v3 dogus is checked and how errors are handled:

| Helm value (`controllerManager.env.`) | Default | Meaning                                                                       |
|---------------------------------------|---------|-------------------------------------------------------------------------------|
| `doguHelmReconciliationInterval`      | `30s`   | Interval in which manual deviations from the resources of a v3 dogu are reverted. |
| `doguHelmRetryInterval`               | `5m`    | Time to wait before retrying a failed installation or upgrade.                |

The values are durations, e.g. `20s`, `2m` or `1h20m`.
Like `doguV3Enabled`, they are set via the component resource of the operator.

## Installing a v3 dogu

To install a v3 dogu, create a dogu resource in the API version `k8s.cloudogu.com/v3beta1` with `spec.doguApiVersion: v3`.
The example installs Nexus from the dogu namespace `official`; `<version>` stands for a version that is available as a v3 dogu.

`nexus.yaml`:

```yaml
apiVersion: k8s.cloudogu.com/v3beta1
kind: Dogu
metadata:
  name: nexus
  labels:
    app: ces
    k8s.cloudogu.com/dogu.name: nexus
spec:
  name: nexus
  doguNamespace: official
  version: <version>
  doguApiVersion: v3
```

```bash
kubectl -n ecosystem apply -f nexus.yaml
```

| Field                 | Meaning                                                                   |
|-----------------------|---------------------------------------------------------------------------|
| `metadata.name`       | Name of the dogu; must match `spec.name`.                                 |
| `spec.name`           | Name of the dogu without dogu namespace.                                  |
| `spec.doguNamespace`  | Namespace of the dogu in the dogu registry, e.g. `official`.              |
| `spec.version`        | Version to install.                                                       |
| `spec.doguApiVersion` | Must be `v3`; without the field the dogu is installed as a v2 dogu.       |

The dogu can be configured right at installation by including `spec.values` or `spec.mappedValues` in the resource.
Both are described in [Configuring v3 dogus](configuring_v3_dogus_en.md).

Notes:

- Always read and edit v3 dogus via `dogus.v3beta1.k8s.cloudogu.com`.
  A plain `kubectl get dogu` uses the API version `v2` and does not show v3 fields such as `values`.
- A migration back from v3 to v2 is not supported.
  The operator rejects it with `Ready=False` and the reason `DowngradeNotAllowed`.
- If `spec.doguApiVersion: v3` is set on an already installed v2 dogu, it is migrated to v3, see [Migrating from v2](migrating_from_v2_to_v3_en.md).

## Checking the installation

The state of the installation is shown in the overview of the dogu resource:

```bash
kubectl -n ecosystem get dogus.v3beta1.k8s.cloudogu.com nexus
```

```text
NAME    SPEC VERSION   INSTALLED VERSION   APP VERSION   READY   HEALTHY   APIVERSION   STOPPED   PAUSE RECONCILIATION   AGE
nexus   <version>      <version>           3.x.y         True    True      v3                                          5m
```

The dogu is installed when `READY` and `HEALTHY` are both `True`.
`READY` shows whether the installation is complete, `HEALTHY` whether all pods of the dogu are running and ready.
During the installation, `READY` is `False`.

If one of them stays `False`, conditions and events name the cause:

```bash
kubectl -n ecosystem get dogus.v3beta1.k8s.cloudogu.com nexus \
  -o jsonpath='{range .status.conditions[*]}{.type}={.status} ({.reason}): {.message}{"\n"}{end}'
kubectl -n ecosystem get events --field-selector involvedObject.kind=Dogu,involvedObject.name=nexus
```

The reason of the condition `Ready` describes the current step or the error, e.g. `Installing` or `DownloadFailed`.
All conditions and events are described in [Conditions and events of v3 dogus](conditions_and_events_en.md).

## Common errors

| Symptom                                                   | Cause / solution                                                                                |
|-----------------------------------------------------------|-------------------------------------------------------------------------------------------------|
| Dogu resource is not processed, operator log `dogu v3 feature flag is disabled` | v3 dogus are not enabled in the operator.                                 |
| Dogu is installed as a v2 dogu                            | `spec.doguApiVersion: v3` is missing.                                                           |
| `Ready=False`, reason `DownloadFailed`                    | Check `dogu-registry-v3`, dogu namespace and version; restart the operator after changing the secret. |
| `Ready=False`, reason `Unauthorized`                      | Check the entry for the chart registry in `ces-container-registries`.                           |
| `Ready=False`, reason `ChartNotFound`                     | The version does not exist as a v3 dogu.                                                        |
| `Ready=False`, reason `SchemaInvalid` or `RenderFailed`   | The configuration does not match the dogu, see [Validation](configuring_v3_dogus_en.md#validation). |
| `Ready=False`, reason `InstallFailed`                     | The installation failed and is retried automatically after `doguHelmRetryInterval`; details are in the message of the condition. |
| `Ready=False`, reason `WorkloadsNotReady`                 | The Helm release is deployed, but the pods of the dogu are not ready; check pods and logs.       |
| `Ready=True`, `Healthy=False`                             | The dogu was ready before and failed afterward, e.g. due to a pod crash, or it is stopped (reason `Stopped`); check pods and logs. |
