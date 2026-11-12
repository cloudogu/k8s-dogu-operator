# v3 dogus

v3 dogus are delivered as Helm charts and installed by the dogu operator.
As with v2 dogus, a dogu resource describes which dogu should be installed in which version.

For administrators, the main changes are:

- **Installation**: The dogu resource is created with `apiVersion: k8s.cloudogu.com/v3beta1` and `spec.doguApiVersion: v3`.
- **Configuration**: Instead of dogu config keys, Helm values are set on the dogu resource (`spec.values`, `spec.mappedValues`).
- **Status**: The state is reported via conditions such as `Ready` and `Healthy`.

## Contents

- [Installing v3 dogus](installing_v3_dogus_en.md): prerequisites, installation and checking the installation
- [Configuring v3 dogus](configuring_v3_dogus_en.md): values, mapped values, log level, global configuration, validation and restart behaviour
- [Stopping, starting and restarting v3 dogus](stopping_starting_and_restarting_v3_dogus_en.md): stopping, starting, pausing and restarting
- [Conditions and events of v3 dogus](conditions_and_events_en.md): reference of all conditions and events
- [Migrating from v2](migrating_from_v2_to_v3_en.md): switching an installed v2 dogu to v3

Developing v3 dogus is described in the [dogu development docs](https://github.com/cloudogu/dogu-development-docs/blob/main/docs/v3/README.md).
