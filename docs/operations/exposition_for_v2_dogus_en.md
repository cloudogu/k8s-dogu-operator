# Exposition for v2 Dogus

This feature controls how the `k8s-dogu-operator` creates `Exposition` custom resources for v2 Dogus.

## Configuration

The feature is controlled by the `EXPOSITION_ENABLED` environment variable.

- `false` (default if the environment variable is unset): no `Exposition` CRs are managed for v2 Dogus
- `true`: the operator creates and updates `Exposition` CRs for v2 Dogus based on their legacy web-route and exposed-port configuration

For Helm deployments, set it in `values.yaml`:

```yaml
controllerManager:
  env:
    expositionEnabled: true
```

## Behavior when `EXPOSITION_ENABLED=true`

When the feature is enabled, the operator no longer uses the legacy service-annotation path for v2 Dogus. Instead, it
creates or updates an `Exposition` CR with the same name as the Dogu during the regular reconciliation flow.

In this mode, the legacy service annotations `k8s-dogu-operator.cloudogu.com/ces-services` and
`k8s-dogu-operator.cloudogu.com/ces-exposed-ports` are no longer written to the Dogu service. External reachability of
the v2 Dogu is described entirely through the `Exposition` CR instead.

For HTTP routes, the operator still reads the familiar legacy input from the image configuration and service setup. In
particular, this includes:

- `SERVICE_TAGS=webapp`
- port-specific definitions such as `SERVICE_<PORT>_TAGS`, `SERVICE_<PORT>_NAME`, `SERVICE_<PORT>_LOCATION`, `SERVICE_<PORT>_PASS`, and `SERVICE_<PORT>_REWRITE`
- `SERVICE_ADDITIONAL_SERVICES`

These values are mapped to `spec.http` of the `Exposition` CR. Existing `location`, `pass`, and legacy rewrite rules
are translated into the HTTP structure used by the `Exposition`.

For dogus with multiple exposed TCP ports, one rule is especially important: if at least one port-specific
`SERVICE_<PORT>_TAGS=webapp` exists, those port-specific tags take precedence over the global `SERVICE_TAGS=webapp`.
This prevents unrelated non-HTTP ports from being treated as web routes.

In addition, exposed ports from `dogu.json` are mapped to layer-4 entries of the `Exposition`:

- `spec.tcp`
- `spec.udp`

Because the legacy exposed-port model does not distinguish the protocol precisely enough for this migration path, each
exposed port currently creates both one TCP and one UDP entry.

Simplified example:

Dockerfile
```Dockerfile
ENV SERVICE_8080_TAGS=webapp
ENV SERVICE_8080_NAME=jenkins
EXPOSE 8080 50000
```

dogu.json
```json
{
  "ExposedPorts": [
    {
      "container": 50000,
      "host": 50000,
      "type": "tcp"
    }
  ]
}
```

This results conceptually in an `Exposition` CR like:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Exposition
metadata:
  name: jenkins
spec:
  http:
    - name: jenkins-8080
      service: jenkins
      port: 8080
      path: /jenkins
  tcp:
    - name: port-50000-50000
      service: jenkins
      port: 50000
      requestedExternalPort: 50000
  udp:
    - name: port-50000-50000
      service: jenkins
      port: 50000
      requestedExternalPort: 50000
```

In this example, only port `8080` is treated as an HTTP route although multiple ports are exposed overall. The
additional port `50000` is mapped only as a layer-4 TCP and UDP entry.

If a v2 Dogu has neither web routes nor exposed ports, the operator removes an already existing `Exposition` CR for
that Dogu again. This avoids leaving empty exposition resources in the cluster.

## Behavior when `EXPOSITION_ENABLED=false`

When the feature is disabled, the operator falls back to the legacy mode for v2 Dogus. This means no `Exposition` CRs
are newly created or updated.

Instead, the operator continues to use the existing service annotations:

- `k8s-dogu-operator.cloudogu.com/ces-services`
- `k8s-dogu-operator.cloudogu.com/ces-exposed-ports`

In this mode, these annotations remain the relevant input for downstream components of the legacy service-discovery and
exposure path.

One important limitation is that turning the flag off does not perform a backward migration. Existing `Exposition` CRs
are neither translated back into legacy annotations nor removed just because the flag was disabled during normal
reconciliation.

Disabling the flag therefore only means:

- from that point on, no new `Exposition` CRs are created or updated for v2 Dogus
- new or updated Dogu services receive the legacy annotations again

Removal of existing `Exposition` CRs still happens only through the normal deletion flow when a Dogu is removed.
