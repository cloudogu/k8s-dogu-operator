# Configuring v3 dogus

v3 dogus are configured via Helm values.
They replace the dogu config keys used to configure v2 dogus.
As an operator, you set the values on the dogu resource, either directly in `spec.values` or via generic keys in `spec.mappedValues`.
In addition, every v3 dogu receives the global configuration of the CES.
The dogu operator merges all sources and rolls out the result.

The examples assume the namespace `ecosystem` and use the Nexus dogu.

## Which values exist?

The values a dogu knows are defined by the dogu itself.
They are listed in the documentation of the dogu and in the following files of its Helm chart:

| File                        | Content                                                            |
|-----------------------------|--------------------------------------------------------------------|
| `values.yaml`               | All values with their defaults                                     |
| `values.schema.json`        | Optional: allowed values and types that are checked against        |
| `dogu-values-metadata.yaml` | Optional: the generic keys for `spec.mappedValues`                 |

## Priority of the values

The operator assembles the values from several sources.
Higher ranks override lower ones:

| Rank | Source                     | Description                                                                   |
|------|----------------------------|-------------------------------------------------------------------------------|
| 0    | `values.yaml` of the chart | Defaults of the dogu                                                          |
| 1    | `global.cesConfig`         | Global configuration of the CES, see [Global configuration](#global-configuration) |
| 2    | Runtime patches            | Values the dogu derives from the global configuration, e.g. a URL from the FQDN |
| 3    | `spec.values`              | Arbitrary overrides of the values                                             |
| 4    | `spec.mappedValues`        | Generic keys that the dogu maps to its values                                 |

When merging, maps are merged recursively.
Lists and plain values, however, are replaced entirely by the source with the higher rank.
An override of a list must therefore always contain the complete list.

## `spec.values`

`spec.values` takes arbitrary overrides of the dogu's values.
Only the values that differ from the default need to be specified; all others keep their default.
The paths follow the structure of the dogu's `values.yaml`.

The following example is for illustration only; the actual paths depend on the dogu:

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
  values:
    nexus:
      configuration:
        normal:
          database:
            maxConnections: "50"
```

On an existing dogu resource, single values can be changed with a merge patch:

```bash
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge -p '
spec:
  values:
    nexus:
      configuration:
        normal:
          database:
            maxConnections: "50"
'
```

### Resetting a value

To use the dogu's default again, remove the override from the dogu resource.
With a merge patch, this is done by setting the value to `null`:

```bash
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge -p '
spec:
  values:
    nexus:
      configuration:
        normal:
          database:
            maxConnections: null
'
```

The merge patch deletes the key from `spec.values`; afterwards the default applies again.

> [!WARNING]
> A `null` that remains in `spec.values` (e.g. via `kubectl apply` of a YAML file with `maxConnections: null`) does not restore the default.
> It is passed on to Helm and removes the key from the defaults as well.
> Therefore, remove values instead of setting them to `null`.

## `spec.mappedValues`

Some settings should be configurable the same way for all dogus, without knowing the structure of the respective dogu's values.
This is what mapped values are for: generic keys such as `mainLogLevel`.
The dogu defines in its `dogu-values-metadata.yaml` which of its values a key affects and which values are allowed.

```yaml
spec:
  mappedValues:
    mainLogLevel: debug
```

- `spec.mappedValues` has the highest rank and also overrides `spec.values`.
- Keys the dogu does not know are ignored without an error message.
  A typo in a key therefore only shows in that nothing changes.

### Log level

The log level of a dogu is set via the key `mainLogLevel`.
It replaces the dogu config key `logging/root` of v2 dogus.
Common values are `debug`, `info`, `warn` and `error`; the dogu's `dogu-values-metadata.yaml` is authoritative.
The dogu translates the value into the notation its application expects, e.g. `debug` into `DEBUG`.

```bash
# set the log level
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge \
  -p '{"spec":{"mappedValues":{"mainLogLevel":"debug"}}}'

# back to the dogu's default
kubectl -n ecosystem patch dogus.v3beta1.k8s.cloudogu.com nexus --type=merge \
  -p '{"spec":{"mappedValues":{"mainLogLevel":null}}}'
```

## Global configuration

The global configuration of the CES (ConfigMap `global-config`) is available to every v3 dogu under `global.cesConfig`.
The dogu can use these values directly or derive its own values from them, e.g. addresses from the FQDN.

- A change to the global configuration is passed on to all v3 dogus immediately.
- Whether it takes effect in the dogu and whether a restart is needed depends on the dogu.
- Stopped or paused dogus only apply the change after the start or the end of the pause.
  Until then, they show `UpdatePending=True`.

## Validation

Before the operator rolls out a change, it checks the assembled values:

1. against the dogu's `values.schema.json`, if present,
2. by rendering the Helm chart as a dry run,
3. for disallowed changes to volumes.

If a check fails, nothing is rolled out and the dogu keeps running with its previous configuration.
The cause is shown as reason on the conditions `Valid` and `Ready`; additionally, there is a warning event:

| Reason                  | Cause                                                  |
|-------------------------|--------------------------------------------------------|
| `SchemaInvalid`         | The values violate the dogu's `values.schema.json`.    |
| `RenderFailed`          | The dogu cannot be rendered with the values.           |
| `VolumeShrink`          | A volume would be shrunk.                              |
| `StorageClassImmutable` | The storage class of a volume would be changed.        |

### Disabling the schema validation

`spec.skipSchemaValidation: true` disables the check against the `values.schema.json`.
The dry-run rendering and the volume checks still take place.
The condition `SchemaValidationSkipped` is then `True`.

> [!WARNING]
> Without schema validation, invalid values only show up during the rollout or in the running dogu.

## Restart on configuration changes

The dogu operator does not restart any pods on configuration changes; it only rolls out the new values.
Whether a change triggers a restart is defined by the dogu:
some settings automatically restart the affected pods, others are picked up by the dogu at runtime or only on the next start.
Which settings trigger a restart is described in the documentation of the dogu.

A manual restart is possible at any time, see [Restarting](stopping_starting_and_restarting_v3_dogus_en.md#restarting).

## What not to do

- **Changing the dogu's resources directly** (ConfigMaps, StatefulSets, ...):
  Such changes are reverted automatically after a short time.
  Configuration is done exclusively via the dogu resource.
- **Editing the dogu resource via the API version `v2`** (`kubectl edit dogu`):
  `values` and `mappedValues` are not visible there.
  Therefore, always edit v3 dogus via `dogus.v3beta1.k8s.cloudogu.com`.

## Checking whether a change has arrived

```bash
kubectl -n ecosystem get dogus.v3beta1.k8s.cloudogu.com nexus
```

As soon as `READY` and `HEALTHY` are `True` again, the change is rolled out.
If `READY` stays `False`, conditions and events name the cause, see [Conditions and events of v3 dogus](conditions_and_events_en.md).
