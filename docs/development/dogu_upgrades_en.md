# Developing Dogu upgrades

This document discusses development decisions related to upgrades of Dogus. 

## Pre-Upgrade Scripts

### Challenge: Difference between file system layout and current user

Copying the pre-upgrade script from the new container to the old container results in a problem if the file 
cannot be copied for rights reasons, such as when the following file system is imagined:

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

Several ways were considered for the solution. The following ways were weighed against each other:

1. the upgrade scripts are always run with the last user specified and their privileges. Copying
   root files with specific users will therefore usually fail.
   - Example: `cp /tmp/dogu-reserved/pre-upgrade.sh / && /pre-upgrade.sh "${versionOld}" "${versionNew}"`
2. since it depends on the script author whether relative or absolute paths are used in the script, it is also not possible to copy the file
   cannot be copied to another location and executed there without risking errors.
   - Example: `cd /tmp/dogu-reserved && ./pre-upgrade.sh "${versionOld}" "${versionNew}"`
3. the same is true for an execution from the working directory of the original script to be run
   - Example: `cd / && /tmp/dogu-reserved/pre-upgrade.sh ${versionOld}" "${versionNew}"`.
4. dynamic insertion of statements in the upgrade script is also discarded, as this solution is complex and error-prone.
   error-prone. It is not easily possible to evaluate and rewrite arbitrary file paths.
   - Example: `sed -i 's|/|/tmp/dogu-reserved|g' /tmp/dogu-reserved/pre-upgrade.sh && /tmp/dogu-reserved/pre-upgrade.sh ${versionOld}" "${versionNew}"`
5. read scripts into a shell pipe and run the stream through a suitable interpreter
   - Example: `cd $(dirname /pre-upgrade.sh) && (cat /tmp/dogu-reserved/pre-upgrade.sh | /bin/bash -s ${versionOld}" "${versionNew}")`

In the end, all solutions have both advantages and disadvantages. However, solutions 2. and 3. offer the least complexity, differing in content only by the working directory. Here the complexity is traded off by conventions, like [pre-upgrade scripts developed](../operations/dogu_upgrades_en.md) must be.
