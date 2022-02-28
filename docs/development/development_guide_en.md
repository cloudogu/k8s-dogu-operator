# Development guide

## Local development

1. Follow the deployment instructions of k8s-ecosystem
2. Edit your `/etc/hosts` and add a mapping from localhost to etcd
    - `127.0.0.1       localhost etcd etcd.ecosystem.svc.cluster.local`
3. port-forward to etcd
```bash
 kubectl port-forward -n=ecosystem etcd-0 4001:2379
```
4. Run `make manifests`
5. Run `make install`
6. export your CES instance credentials for the operator to use
    - `export CES_REGISTRY_USER=instanceId && export CES_REGISTRY_PASS='instanceSecret'`
7. Run `make run` to run the dogu operator locally

This document contains information about all make targets used for this controller.

## Target Overview (make)

The command `make help` prints all available targets and their descriptions in the command line.