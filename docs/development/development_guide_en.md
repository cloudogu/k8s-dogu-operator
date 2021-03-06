# Development guide

## Local development

1. Follow the deployment instructions of k8s-ecosystem
2. Edit your `/etc/hosts` and add a mapping from localhost to etcd
    - `127.0.0.1       localhost etcd etcd.ecosystem.svc.cluster.local`
3. Open the file `.env.template` and follow the instructions to create an env file with your personal data
4. Make an etcd port forward
   - `kubectl -n=ecosystem port-forward etcd-0 4001:2379`
5. Run `make run` to run the dogu operator locally
6. Delete any existing dogu operator deployments to avoid concurrency issues
   - `kubectl -n=ecosystem delete deployment k8s-dogu-operator`

## Makefile-Targets

The command `make help` prints all available targets and their descriptions in the command line.

## Local image build

To build the image of the `dogu-operator` locally a `.netrc` file is needed in the project directory.

```
machine github.com
login <username>
password <token>
```

The token needs permissions to read private repositories.

## Using custom dogu descriptors

The `dogu-operator` is able to use a custom `dogu.json` for a dogu during installation.
This file must be in the form of a configmap in the same namespace. The name of the configmap must be `<dogu>-descriptor`
and the user data must be available in the data map under the entry `dogu.json`.
There is a make target to automatically generate the configmap - `make install-dogu-descriptor`.
Note that the file path must be exported under the variable `CUSTOM_DOGU_DESCRIPTOR`.

After a successful Dogu installation, the ConfigMap is removed from the cluster.

## Filtering the Reconcile function

So that the reconcile function is not called unnecessarily, if the specification of a dogu does not change,
the `dogu-operator` is started with an update filter. This filter looks at the field `generation` of the old
and new dogu resource. If a field of the specification of the dogu resource is changed the K8s api increments
`generation`. If the field of the old and new dogu is the same, the update is not considered.
