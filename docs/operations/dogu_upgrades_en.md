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
This uses the updated image of the Dogus and copies only the script into the old container.
A designated volume is already created during the installation.

### Challenge: Difference between file system layout and current user.

Copying the pre-upgrade script from the new to the old container results in a problem if the file cannot be copied due to
cannot be copied for permissions reasons, such as when the following file system is imagined:

```
ls -lha / 
drwxr-xr-x 1 root root 4.0K Dec 13 10:47 .
-rwxrwxr-x 1 root root 704 Oct 12 14:25 pre-upgrade.sh
...

ls -lha /tmp/dogu-reserved/
drwxrwsr-x 3 root doguuser 1.0K Dec 13 10:48 .
-rwxr-xr-x 1 doguuser doguuser 704 Dec 13 10:48 pre-upgrade.sh
...
```

Several paths were considered for the solution. The following four ways were weighed and found to be too problematic:

1. the upgrade scripts are always executed with the last specified user and its rights. Copying root files with specific users will usually fail.
   - incorrect example: `cp /tmp/dogu-reserved/pre-upgrade.sh / && /pre-upgrade.sh "${versionAlt}" "${versionNeu}"`
2. since it depends on the script author whether relative or absolute paths are used in the script, it is also not possible to copy the file cannot be copied to another location and executed there without risking errors.
   - incorrect example: `cd /tmp/dogu-reserved && ./pre-upgrade.sh "${versionAlt}" "${versionNeu}"`
3. the same applies to an execution from the working directory of the original script to be started
   - incorrect example: `cd / && /tmp/dogu-reserved/pre-upgrade.sh`.
4. a dynamic introduction of statements in the upgrade script is also rejected, this solution on the one hand is complex and error-prone. It is not easily possible to evaluate and rewrite arbitrary file paths.
   - incorrect example: `sed -i 's|/|/tmp/dogu-reserved|g' /tmp/dogu-reserved/pre-upgrade.sh && /tmp/dogu-reserved/pre-upgrade.sh`

Instead, the following solution was chosen:

This consists of changing to the directory for which the upgrade script was designed. Then the contents of the script is executed by shell piping directly through the chosen script interpreter. This behavior has been implemented by the Dogu operator. It is rather interesting for Dogu developers to look at the design of their own container in this respect.

- This snippet can be used to test this behavior in the old Dogu container:
- Test example: `sh -c "cd (basename /preupgrade.sh) && sh -c < /tmp/dogu-reserved/pre-upgrade.sh"`.
   - here the second occurrence of the shell interpreter `sh` has to be replaced by one defined in the script to ensure maximum compatibility of script and interpreter.

### Restrictions

As a result of the described behavior, the following restrictions apply to pre-upgrade scripts:

- The SetUID bit cannot currently be used for pre-upgrade scripts because it is not lost by `cp`.
- `/bin/cp` must be installed
- `/bin/grep` must be installed in case the pre-upgrade script or its directory has a different
  Unix user than the one present in the running dogu.
- It is assumed that the pre-upgrade script is a shell script and not any other
  executable (e.g. a Linux binary file).
   - If this is not the case, the container image must be structured in such a way that the copy process can be executed with the
     container user and the execution of the executable is possible.

## Upgrade special cases

### Downgrades

Downgrades of Dogus are problematic if the new Dogu version modifies the data basis of the older version by the upgrade in such a way that the older version can no longer do anything with the data. **Under certain circumstances, the Dogu thus becomes incapable of working**. Since this behavior depends very much on the tool manufacturer, it is generally not possible to _downgrade_ Dogus.

Therefore the dogu operator refuses to upgrade a dogu resource to a lower version. This behavior can be disabled by using the `spec.upgradeConfig.forceUpgrade` switch with a value of True.

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
