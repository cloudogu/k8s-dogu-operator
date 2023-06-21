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
    dogu.name: my-dogu
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
    dogu.name: my-dogu
    app: ces
spec:
  name: official/my-dogu
  version: 1.2.3-5
```

## Pre-upgrade scripts

For the pre-upgrade script, a pod is started during the upgrade process.
This uses the updated image of the Dogu and copies only the script named in the Dogu.json to the old
container. It is then executed in the old Dogu during runtime. This is done from the same path where the script was in the new Dogu.

### Requirements for a pre-upgrade script

This section defines easy-to-implement requirements for Dogu developers to enable the execution of
pre-upgrade scripts as error-free and transparent as possible.

#### Parameters

Pre-upgrade scripts must take exactly two parameters:

1. the old Dogu version that is currently running
2. the new Dogu version to which the upgrade should be applied.

Based on this information, pre-upgrade scripts can make crucial decisions. This can include:
- denial of upgrades for version jumps that are too large
- adjusted preparation measures per found version

For example, the pre-upgrade script could be called like this:

```bash
/path/to/pre-upgrade.sh 1.2.3-4 1.2.3-5
```

There is no provision for passing any other parameters.

#### Use of absolute file references

When it comes to file processing, pre-upgrade scripts must use absolute file paths,
since there is no way to ensure that a script will always be called from its source location.

#### Do not use other files

Pre-upgrade scripts are copied from the upgrade image to the Dogu container to be executed there.
Since only the pre-upgrade script and unrelated files can be named in the Dogu descriptor `dogu.json`,
a pre-upgrade script must be fully constructed in its functional scope.

This excludes in particular the shell sourcing of other files, since here frequently wrong assumptions of version levels lead to errors.

#### Executability

- The SetUID bit cannot currently be used for pre-upgrade scripts because it is lost by copying the script from pod to pod (using `tar`).
- `/bin/tar` must be installed
- It is assumed that the pre-upgrade script is a shell script and not any other
  executable (e.g. a Linux binary).
   - If this is not the case, the container image must be structured in such a way that the copy operation can be executed with the
     container user and the execution of the executable is possible.
- The pre-upgrade script is executed by the current container user in the old dogu

#### Limitations

The size of the pre-upgrade script is only limited by the RAM (random access memory).

## Post-Upgrade Script

Unlike the pre-upgrade script, the post-upgrade script is subject to only minor constraints because the script is usually already in its execution location.
The post-upgrade script is executed in the new dogu at the end of the upgrade process.
The dogu is responsible for waiting for the post-upgrade script to finish.
This is where the use of the dogu state has proven helpful:

```bash
# post-upgrade.sh
doguctl state "upgrading
# upgrade routines go here...
doguctl state "starting
```

```bash
# startup.sh
while [[ "$(doguctl state)" == "upgrading" ]]; do
  echo "Upgrade script is running. Waiting..."
  sleep 3
done
# regular start-up goes here
```

After that the upgrade is finished.

## Upgrade special cases

### Downgrades

Downgrades of Dogus are problematic if the new Dogu version modifies the data basis of the older version by the upgrade in such a way
that the older version can no longer do anything with the data. **Under certain circumstances, the Dogu thus becomes incapable of working**.
Since this behavior depends very much on the tool manufacturer, it is generally not possible to _downgrade_ Dogus.

Therefore, the dogu operator refuses to upgrade a dogu resource to a lower version.
This behavior can be disabled by using the `spec.upgradeConfig.forceUpgrade` switch with a value of True.

**Caution possible data corruption:**
You should clarify beforehand that the dogu will not be damaged by the downgrade.

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: cas
  labels:
    dogu.name: cas
    app: ces
spec:
  name: official/cas
  version: 6.5.5-3
  upgradeConfig:
    # for downgrade from v6.5.5-4
    forceUpgrade: true
```

### Dogu namespace change

A dogu namespace change is made possible by changing the dogu resource. This may be necessary, for example, when a new dogu is published to a different namespace.

This behavior can be disabled by using the switch `spec.upgradeConfig.allowNamespaceSwitch` with a value of `true`.

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: cas
  labels:
    dogu.name: cas
    app: ces
spec:
  name: official/cas
  version: 6.5.5-4
  upgradeConfig:
    allowNamespaceSwitch: true
```
