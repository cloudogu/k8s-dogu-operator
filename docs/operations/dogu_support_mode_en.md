# Support strategy for Dogus

Under certain conditions it can happen that the pods of the Dogus are in a restart loop.
In such cases it is helpful to connect to the container via shell and analyze the filesystem.
The support mode stops the restart loop and puts the dogu into a "frozen" mode to allow the connection to the container.

## Enabling the support mode

The Dogu resource has a boolean field `supportMode` in its description.
To put a Dogu in support mode, this must be set to `true` by updating the Dogu resource.

Example:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: postfix
  annotations:
    test: dev
  labels:
    dogu.name: postfix
    app: ces
spec:
  name: official/postfix
  version: 3.6.4-3
  supportMode: true
```

`kubectl apply -f postfix.yaml`

This ignores other changes to the dogu description. Also, if a dogu is already in the support mode.

Technically, the `k8s-dogu-operator` updates the deployment of the dogu. The startup command of the container is
ignored by adding a sleep command. The ordinary probes of the container are deleted so that in the
maintenance, the container is not restarted by the pod controller. In addition, an environment variable `SUPPORT_MODE` 
`true` is added to the container. After updating the deployment, the pods of the Dogus are restarted and one can connect to them.

Example:

`k exec -it postfix-<pod_id> -- sh`

## Deactivating the support mode

To restore a dogu to its initial state, the dogu resource has to be updated with `supportMode` `false`.

Example:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: postfix
  annotations:
    test: dev
  labels:
    dogu.name: postfix
    app: ces
spec:
  name: official/postfix
  version: 3.6.4-3
  supportMode: false
```

`kubectl apply -f postfix.yaml`
