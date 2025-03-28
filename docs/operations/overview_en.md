# Dogu operator and custom resource definition `Dogu`.

A controller is a Kubernetes application that is informed about state changes of resources that it listens for. Since the [operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) is used for this, they are also called _operators_.

Such operators often come into play in the context of _Custom Resource Definitions_ (CRD) when Kubernetes is to be extended with custom resource types. The Dogu operator is such an operator, which takes care of the management of Dogus in terms of Kubernetes resources.

The basic idea of the operator is relatively simple. It takes care of successful execution of Dogu in a cluster. The operator specifies the resources to be used based on these information:
- dogu.json
- container image
- CES instance credential
  in order to create all required Kubernetes resources, e.g.:
   - Container
   - Persistent Volume Claim
   - Persistent Volume
   - Service
   - Ingress

Each of the Kubernetes resources must be created by a description (usually in YAML format). Because of the amount of resources and the amount of properties per resource, a Dogu installation quickly becomes tedious and error-prone. The Dogu operator provides useful support here by automatically taking care of resource management in Kubernetes. With a few lines of dogu description, a dogu can be installed like this (see below).

The following graphic shows different tasks during a Dogu installation.

![PlantUML diagram: k8s-dogu-operator installs a dogu](figures/k8s-dogu-operator-overview.png
"k8s-dogu-operator installs a dogu.")

## Dogu management

The CRD (Custom Resource) description for Dogus looks something like this:

Example: `ldap.yaml`
```yaml
apiVersion: k8s.cloudogu.com/v2
kind: Dogu
metadata:
  name: ldap
  labels:
    dogu.name: ldap
    app: ces
spec:
  name: official/ldap
  version: 2.4.48-3
```

> [!IMPORTANT]
> `metadata.name` and the simple name of the dogu in `spec.name` must be equal.
> The simple name is the part after the slash (`/`), so without the dogu namespace.
> For example, for a dogu with `spec.name` of `k8s/nginx-ingress` the `metadata.name` of `nginx-ingress` would be ok, while `nginx` would not.

To install the LDAP dogu, a simple call is enough:

```bash
kubectl apply -f ldap.yaml
```

With the introduction of the Dogu CRD, Dogus we can use native Kubernetes resources, for example:

```bash
# lists a single dogu
kubectl get dogu ldap
# lists all installed dogus
kubectl get dogus
# delete a single dogu
kubectl delete dogu ldap
```

## dogu operator vs `cesapp`

In terms of their function, dogu operator and `cesapp` are very comparable because both take care of managing and orchestrating dogus in their respective execution environments.

However, in the long run, the Dogu operator will not reach the size and complexity of `cesapp` because its function is very much related to installing, updating and uninstalling Dogus.

## Kubernetes volumes vs Docker volumes.

With few exceptions, Dogus often define volumes in their `dogu.json` where their state should be persisted. In the previous EcoSystem, this was solved by Docker volumes that the `cesapp` set up and assigned to the container during installation.

In Kubernetes, persistence is more decoupled. A Persistent Volume Claim (PVC) defines the size of the desired volume, which in turn is a persistent volume provisioned by a storage provider.

Unlike a Docker volume, a Kubernetes volume cannot easily resize because it is immutable. In addition, separate processes may be started for each Kubernetes volume, which again consume main memory.

The dogu operator creates a single volume for these reasons. All volumes defined in `dogu.json` are then mounted as subdirectories in the volume.
