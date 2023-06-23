# Configuring resource requests for Dogus

This document describes how to configure and apply resource requirements (limits and requests) for a Dogu.

- **Resource Requests:** Specify the minimum resources (CPU cores, memory, ephemeral storage) required by a Dogu for it to be functional.
  The Kubernetes scheduler ensures that the Dogu is started on a node with sufficient resources.
- **Resource Limits:** Specify the maximum amount of resources a Dogu is allowed to use.
  If the CPU core limit is exceeded, the container runtime throttles the available CPU resources for the pod.
  If the memory limit or ephemeral storage is exceeded, the respective pod is "evicted" and restarted.

## Prerequisites

* An operational Cloudogu MultiNode EcoSystem.

## Limits & Requests

Resource requirements can be applied to each Dogu.
In general, there are three different limits or requests:

1. **CPU cores**: More information can be found on the official page of
   Kubernetes: [Kubernetes-CPU](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-cpu).
2. **Memory**: More information is available on the official page of
   Kubernetes: [Kubernetes-Memory](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory)
3. **Ephemeral-Storage**: More information is available on the official site of
   Kubernetes: [Kubernetes-Ephemeral-Storage](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#local-ephemeral-storage)

## Configuring resource requests.

Limits and requests are generally configured in etcd. **Note:** Setting a resource requirements does not automatically result in a
restart of the Dogus. It must be applied explicitly. This is described in the next section.

In general, the following entries can be set in any `config` section of a Dogu under the `container_config` section:

**CPU cores**

- key for request: `config/<DOGU_NAME>/container_config/cpu_core_request`
- key for limit: `config/<DOGU_NAME>/container_config/cpu_core_limit`
- Optional
- Description: Sets the CPU resource requirement for each started pod of the Dogu.
- Format:
  See https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes

**Memory

- Key for request: `config/<DOGU_NAME>/container_config/memory_request`
- Key for limit: `config/<DOGU_NAME>/container_config/memory_limit`
- Optional
- Description: Sets the memory resource requirement for each started pod of the Dogu.
- Format: The configurable values for the keys are each a string of the form `<number value><unit>` and describe the maximum amount of memory usable by the Dogu.
  Note here that there must be no space between the numeric value and the unit.
  Available units are `b`, `k`, `m` and `g` (for byte, kibibyte, mebibyte and gibibyte).
  **Note:** This does not use the format used by Kubernetes!

**Ephemeral-Storage**

- Key for request: `config/<DOGU_NAME>/container_config/storage_request`
- key for limit: `config/<DOGU_NAME>/container_config/storage_limit`
- Optional
- Description: sets the ephemeral storage resource requirement for each started pod of the Dogu.
- Format: The configurable values for the keys are each a string of the form `<number value><unit>` and describe the maximum amount of storage that can be used by the Dogu.
  Note here that there must be no space between the numeric value and the unit.
  Available units are `b`, `k`, `m` and `g` (for byte, kibibyte, mebibyte and gibibyte).
  **Note:** This does not use the format used by Kubernetes!


## Apply configured resource requests

In order for the resource requirements to be applied, the global etcd key: `config/_global/sync_resource_requirements` 
must be created/modified/deleted. Any change to the key will result in the start of an automatic update process
for all Dogus. In this update process the resource requirements are applied to all Dogus and the Dogus are restarted if new resource requirements are set.
Unchanged Dogus are not restarted. In general, the update process can be started with this command:

```bash
etcdctl set /config/_global/sync_resource_requirements true
```