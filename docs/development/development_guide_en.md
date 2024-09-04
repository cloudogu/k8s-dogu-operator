# Development guide

## Local development

1. Run `make run` to run the dogu operator locally

### Debugging with IntelliJ

1. Follow steps above except `make run`
2. Use the IntelliJ-section of the .env-template
3. print your set of env-variables with `make print-debug-info`
4. copy the result in your intelliJ run configuration as environment
5. start main.go in debug-mode

## Makefile-Targets

The command `make help` prints all available targets and their descriptions in the command line.

## Using custom dogu descriptors

The `dogu-operator` is able to use a custom `dogu.json` for a dogu during installation.
This file must be in the form of a configmap in the same namespace. The name of the configmap must be `<dogu>-descriptor`
and the user data must be available in the data map under the entry `dogu.json`.
There is a make target to automatically generate the configmap - `make install-dogu-descriptor`.

After a successful Dogu installation, the ConfigMap is removed from the cluster.

## Filtering the Reconcile function

So that the reconcile function is not called unnecessarily, if the specification of a dogu does not change,
the `dogu-operator` is started with an update filter. This filter looks at the field `generation` of the old
and new dogu resource. If a field of the specification of the dogu resource is changed the K8s api increments
`generation`. If the field of the old and new dogu is the same, the update is not considered.
