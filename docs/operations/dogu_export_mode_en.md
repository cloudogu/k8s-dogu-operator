# Export mode for Dogus

To migrate a multinode CES instance, the data of all Dogus must be copied from the source instance to the target instance.
To minimize downtime during the migration, the data should be copied while the source instance is still in operation.
The “Export mode” of a Dogu makes its data volume available for migration via an “Exporter” sidecar container.

## Activating the export mode

The Dogu resource has a boolean field `exportMode` in its description.
To set a Dogu to export mode, this must be set to `true` by updating the Dogu resource.

Example:
```yaml
apiVersion: k8s.cloudogu.com/v2
kind: Dogu
metadata:
  name: ldap
  annotations:
    test: dev
  labels:
    dogu.name: ldap
    app: ces
spec:
  name: official/ldap
  version: 2.6.8-3
  exportMode: true
```

`kubectl apply -f ldap.yaml`

> **Note:** Activating the export mode for a restart of the Dogus.

Technically, the `k8s-dogu-operator` updates the deployment of the Dogus.
An additional “sidecar” container is added to the Dogus pod.
This container also has a volume mount for the data volume of the Dogus.
This makes the data available for migration via “Rsync over SSH”

## Deactivation of the export mode

To restore a Dogu to its original state, the Dogu resource must be updated with the `exportMode` field  to `false`.

Example:
```yaml
apiVersion: k8s.cloudogu.com/v2
kind: Dogu
metadata:
  name: ldap
  annotations:
    test: dev
  labels:
    dogu.name: ldap
    app: ces
spec:
  name: official/ldap
  version: 2.6.8-3
  exportMode: false
```

`kubectl apply -f ldap.yaml`

The additional sidecar container is now removed again.

**Note:** The deactivation of the export mode for a restart of the Dogus.
