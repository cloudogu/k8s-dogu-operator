# Development guide

## Local development

1. Follow the deployment instructions of k8s-ecosystem
2. Edit your `/etc/hosts` and add a mapping from localhost to etcd
    - `127.0.0.1       localhost etcd etcd.ecosystem.svc.cluster.local`
3. port-forward to etcd (process blocks and should execute in a new terminal)
```bash
 kubectl port-forward -n=ecosystem etcd-0 4001:2379
```
4. Run `make manifests`
5. Run `make install`
6. export your CES instance credentials for the operator to use
    - `export CES_REGISTRY_USER=instanceId && export CES_REGISTRY_PASS='instanceSecret'`
7. export your CES instance namespace
   - `export NAMESPACE=ecosystem`
8. Run `make run` to run the dogu operator locally

## Makefile-Targets

The command `make help` prints all available targets and their descriptions in the command line.

## Using custom dogu descriptors

The `dogu-operator` is able to use a custom `dogu.json` for a dogu during installation.
This file must be in the form of a configmap in the same namespace. The name of the configmap must be `<dogu>-descriptor`
and the user data must be available in the data map under the entry `dogu.json`.
There is a make target to automatically generate the configmap - `make generate-dogu-descriptor`.
Note that the file path must be exported under the variable `CUSTOM_DOGU_DESCRIPTOR`.

After a Dogu installation the Dogu is entered as owner in the ConfigMap. If you uninstall the Dogu afterwards
the ConfigMap is also removed from the cluster.