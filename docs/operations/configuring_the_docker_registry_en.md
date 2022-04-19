# Configuring the Docker Registry

This document describes how to attach the required Docker Registry to the `k8s-dogu-operator`.
can be connected.

## Requirements

* A K8s cluster must be present. This should be able to be accessed via `kubectl`.

## Dogu Registry

The Docker Registry is a storage system for the images of the Dogus. This registry contains the images
about all published Dogus and serves thus as starting point for the Dogu operator.

For a Docker Registry to be attached, a secret must be created in the K8s cluster.
must be created. This secret contains the login information needed for the `k8s-dogu-operator`:

**Example**

Username: mydockerlogin
Password: mydockerpassword

## Create Docker Registry Secret

The secret containing the Docker Registry data must be created under the name `k8s-dogu-operator-docker-registry`. The
Registry data will be encrypted as literals in the secret. A correct secret can be created with `kubectl`.
as follows:

```bash
kubectl --namespace <cesNamespace> create secret generic k8s-dogu-operator-docker-registry \
--from-literal=username="myusername" \
--from-literal=password="mypassword"
```

After that the `k8s-dogu-operator` can be [installed](installing_operator_into_cluster_en.md) as usual.