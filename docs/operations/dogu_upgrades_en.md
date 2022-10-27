# Dogu upgrades

At first glance, a dogu upgrade represents nothing more than importing a new dogu version into the Cloudogu EcoSystem.
A dogu upgrade is one of several operations that `k8s-dogu-operator` supports. Basically, it is only possible to upgrade
dogus with a higher version. Special cases are discussed in the section "Upgrade special cases".

Such an upgrade can be easily created.

**Example:**

A dogu has already been installed in an older version with this dogu resource using `kubectl apply`:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: my-dogu
  labels:
    dogu: my-dogu
    app: ces
spec:
  name: official/my-dogu
  version: 1.2.3-4
```

Upgrading the Dogus to version `1.2.3-5` is very simple. Create a comparable resource with a newer version and apply it
to the cluster again with `kubectl apply ...`:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: my-dogu
  labels:
    dogu: my-dogu
    app: ces
spec:
  name: official/my-dogu
  version: 1.2.3-5
```

## Pre-upgrade scripts

For the pre-upgrade script, a pod is started during the upgrade process.
This uses the updated image of the Dogus and copies only the script into the old container.
A designated volume is already created during the installation.

## Upgrade special cases

### Downgrades

To Do DG

### Change of a Dogu namespace

To Do DNW
