# Configuration of the Dogu Registry

This document describes how to connect the required Dogu Registry to the `k8s-dogu-operator`.
can be connected.

## Requirements

* A K8s cluster must be available. This should be controllable via `kubectl`.

## Dogu Registry

The Dogu Registry is a storage system for Dogus. This registry contains information about all
published Dogus and thus serves as a contact point for the Dogu operator.

In order to connect a custom dogu configuration, a secret must be created in the K8s cluster.
must be created. This secret contains the endpoint and the login information needed for the `k8s-dogu-operator`:

**Example**

Registry endpoint (API V2): https://my-registry.com/api/v2/
Username: myusername
Password: mypassword

## Create Dogu Registry Secret

The secret containing the Dogu registry data must be created under the name `k8s-dogu-operator-dogu-registry`. The
Registry data is stored as literals encrypted in the Secret. A correct secret can be created with `kubectl` as follows
as follows:

```bash
kubectl --namespace <cesNamespace> create secret generic k8s-dogu-operator-dogu-registry \
--from-literal=endpoint="https://my-registry.com/api/v2" \
--from-literal=username="myusername" \
--from-literal=password="mypassword"
```

After that the `k8s-dogu-operator` can be [installed](installing_operator_into_cluster_en.md) as usual.