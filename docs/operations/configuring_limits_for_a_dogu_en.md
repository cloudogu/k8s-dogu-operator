# Configuring physical limits for dogus

This document describes how to configure and apply physical limits for a dogu.

## Prerequisites

* An operational Cloudogu MultiNode EcoSystem.

## Physical Limits

Physical limits can be applied to each dogu and restrict the pod of the dogu to specified limits.
limits.
Generally, there are three different limits:

1. **CPU Limit**: More information is available on the official page of
   Kubernetes: [Kubernetes-CPU](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-cpu)
1. **Memory Limit**: More information is available on the official page of
   Kubernetes: [Kubernetes-Memory](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory)
1. **Ephemeral-Limit**: More information is available on the official page of
   Kubernetes: [Kubernetes-Ephemeral-Storage](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#local-ephemeral-storage)

## Configuring Limits

Limits are generally configured in etcd. **Note:** Setting a limit does not automatically cause a restart of the
restart of the Dogus. Limits must be applied explicitly. This is described in the next section.

In general, the following entries can be set in any `config` section of a Dogus under the `pod_limit` section:

**CPU-Limit**

- Key: `config/<DOGU_NAME>/pod_limit/cpu`
- Optional
- Description: Sets the CPU limit for each started pod of the Dogus.
- Format:
  See https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes

**Memory limit**

- Key: `config/<DOGU_NAME>/pod_limit/memory`
- Optional
- Description: Sets the memory limit for each started pod of the Dogus.
- Format:
  See https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes

**Ephemeral-Storage-Limit**.

- Key: `config/<DOGU_NAME>/pod_limit/ephemeral_storage`
- Optional
- Description: Sets the ephemeral storage limit for each started pod of the dogus.
- Format:
  See https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes

## Apply configured limits

In order for the limits to be applied, the global etcd key: `config/_global/trigger_container_limit_sync`
must be created/modified/deleted. Any change to the key will result in the start of an automatic update process
for all dogus. In this update process the limits are applied to all dogus and the dogus are restarted if new limits are set.
are restarted. Unchanged dogus are not restarted. In general, the update process can be started with the
command:

```bash
etcdctl set /config/_global/trigger_container_limit_sync true
```