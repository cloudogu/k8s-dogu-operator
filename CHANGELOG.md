# k8s-dogu-operator Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.27.0] - 2023-03-27
### Added
- [#87] Support for Split-DNS environments

## [v0.26.1] - 2023-03-03
### Fixed
- [#85] Fix DoS vulnerability by upgrading the k8s controller-runtime (along with `k8s-apply-lib`)

## [v0.26.0] - 2023-02-23
### Added
- The dogu operator can now handle existing private dogu keys.
### Changed
- [#83] Stabilize the dogu registration process.
  - Dogus only will be enabled last in the registration process to prevent faulty states in error cases.
### Fixed
- [#79] Fix a bug where an installation failed if old PVCs stuck with terminating status.

## [v0.25.0] - 2023-02-17
### Added
- [#81] Add optional volume mounts for selfsigned certs of the docker and dogu registries.

## [v0.24.0] - 2023-02-08
### Added
- [#78] Add a no spam filter to process every event thrown by the controller.
### Fixed
- [#76] Fix an issue where an update of a deployment in the dogu upgrade process lead to a resource conflict.

## [v0.23.0] - 2023-02-06
### Added
- [#74] Add init container for dogus with volumes to execute chown on the directories with the specified uid and gid.

## [v0.22.0] - 2023-01-31
### Changed
- [#72] Remove the service environment variables from dogu pods with `enableServiceLinks: false` in the podspec of
  the dogu pods. Cluster-aware dogus are generally discouraged to use service link env vars because of security considerations. Instead, the service DNS names should be used to address these services as described in the [Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/service/#dns).
- Update makefiles to version 7.2.0.
- Update ces-build-lib to 1.62.0.

## [v0.21.0] - 2023-01-11
### Changed
- [#70] add/update label for consistent mass deletion of CES K8s resources
  - select any k8s-dogu-operator related resources like this: `kubectl get deploy,pod,dogu,rolebinding,... -l app=ces,app.kubernetes.io/name=k8s-dogu-operator`
  - select all CES components like this: `kubectl get deploy,pod,dogu,rolebinding,... -l app=ces`

## [v0.20.0] - 2023-01-06
### Added
- Added kubernetes client for handling dogu resources of a cluster.
### Fixed
- Accept kind `ces` for ces control service accounts.

## [v0.19.0] - 2022-12-22
### Added
- [#44] Support for expanding dogu volumes. For details see [volume expansion docs](docs/operations/expand_volume_en.md).

## [v0.18.1] - 2022-12-20
### Fixed
- [#66] Fixes dogu upgrade problems of `official/scm` dogus and add a fallback strategy to execute pre-upgrade scripts
  - for a detailed discussion please see the [dogu upgrade docs](docs/operations/dogu_upgrades_en.md).
- Fixes a nil pointer panic when upgrading Dogus without `state` health check

## [v0.18.0] - 2022-12-01
### Added
- [#61] Add the yaml of the Dogu CRD in api package. Other controllers/operators can consume it for e.g. integration
  tests with envtest. The `generate` make target will refresh the yaml.

## [v0.17.0] - 2022-11-24
### Fixed
- [#62] Fix wrong exposed service object key. During the creation of exposed services some wrong object keys are used. 
  Later-on this leads to an error when tried to get these resources.
- [#64] Fix the creation of service annotations by ignoring all irrelevant environment variables and by correctly
  splitting environment variables containing multiple `=`.

## [v0.16.0] - 2022-11-18
### Added
- [#59] Support for extended volume definitions in the `dogu.json`, allowing the creation of kubernetes specific 
  volumes.
- [#59] Support for extended service account definitions in the `dogu.json`, allowing the creation of kubernetes 
  accounts for dogus.

### Removed
- [#59] Mechanism to patch the generated dogu deployment with custom volumes and service account names. These are now
  supported by the `dogu.json` and natively generated into the deployment.

## [v0.15.0] - 2022-11-15
### Changed
- [#55] Refactoring the creation and update of kubernetes dogu resources.
- Extract interfaces and mocks to an internal package, which removes duplicate interfaces and avoids import cycles.

## [v0.14.0] - 2022-11-09
### Added
- [#48] Make dogu registry URL schema configurable.
- [#47] Execute Dogu pre-upgrade scripts in upgrade process. See [dogu upgrades](docs/operations/dogu_upgrades_en.md).
- [#51] Execute Dogu post-upgrade scripts in upgrade process. See [dogu upgrades](docs/operations/dogu_upgrades_en.md).

### Removed
- [#52] Remove cesapp dependency and use cesapp-lib.

## [v0.13.0] - 2022-10-12
### Added
- [#43] Dogu resource has now a support mode, which leads the dogu pods to a freeze but running state.
  This is useful in cases where the dogu is in a restart loop. See [support mode](docs/operations/dogu_support_mode_en.md)
  for more information.

## [v0.12.0] - 2022-09-29
### Added
- [#41] Fire events to the specific dogu resource when installing or deleting a dogu. See 
[event policy](docs/development/event_policy_for_the_operator_en.md) for more information.
- [#40] Support dogu upgrades
  - `k8s-dogu-operator` checks the dogu health and all its dependencies similar to the `cesapp`
  - The current PVC handling ignores any changes for dogu upgrades. This issue will be solved later.
  - for more information about requeueing and internal error handling [the docs on reconciliation](docs/development/reconciliation_en.md)
    provide more insights

### Fixed
- fixes a possible parsing error when the environment variable `LOG_LEVEL` is set but empty

### Changed
- [#41] Update makefiles to version `v7.0.1`.

## [v0.11.0] - 2022-08-29
### Changed
- [#36] Update `cesapp-lib` to version `v0.4.0`
- [#36] Update `k8s-apply-lib` to version `v0.4.0`
- [#36] Changed the loggers for the both libs `cesapp-lib` and `k8s-apply-lib` according to the new logging interface.

## [v0.10.0] - 2022-08-25
### Changed
- [#38] Detect existing PVC when installing a dogu. This allows users to store initial data for dogus before their 
installation. See [documentation](docs/operations/edit_dogu_volume_data_en.md) for more details.
- [#38] Update `ces-build-lib` to version 1.56.0
- [#38] Update `makefiles` to version 6.3.0

## [v0.9.1] - 2022-07-18
### Fixed
- [#34] Fixed a permission issue where the remote registry trys to write to a non-privileged cache dir.

## [v0.9.0] - 2022-07-13
### Added
- [#28] Dogu hardware limit updater responsible to update the deployments of dogus with configured container limits.

### Changed
- [#28] Updated cesapp-lib to version 0.2.0
- [#29] Remove implementation of the remote http dogu registry and instead, reuse the implementation from the cesapp-lib.
- [#31] Split dogu manager in separate components according to it functions (install, update, delete).

## [v0.8.0] - 2022-06-08
### Added
- [#26] Allow the definition of custom Deployment in dogus. In such custom Deployments it is possible
  to define extra volumes,volume mounts, and the used service account for the dogu Deployment.

## [v0.7.0] - 2022-06-07
### Added
- [#6] Installing generic kubernetes resources when installing a dogu. These resources need to be provided by the dogu 
image at the `k8s` folder in the root path (`/k8s`):
  - There are no restriction for namespaced resources.
  - The creation of cluster scoped resources is restricted and also their 
  deletion is not performed automatically as they could be used inside multiple namespaces. 

### Changed
- [#6] Update makefiles to version 6.0.2

## [v0.6.0] - 2022-05-24
### Added
- [#19] Remove service account on dogu deletion.

## [v0.5.0] - 2022-05-23
### Added
- [#20] Detect and write encrypted configuration entries for dogus into the etcd registry when installing a dogu.

## [v0.4.0] - 2022-05-12
### Added
- [#15] Add startup probe based on state at dogu deployment generation
- [#15] Add liveness probe based on tcp port at dogu deployment generation

## [v0.3.1] - 2022-05-12
### Fixed
- [#17] Requeue dogu installation when an error occurs when creating a dependent service account.

## [v0.3.0] - 2022-05-03
### Added
- [#2] Annotation `k8s-dogu-operator.cloudogu.com/ces-services` to Dogu-`Services` containing information of
  related CES services. For more information see [Annotations](/docs/operations/annotations_en.md).
- [#13] The automatic generation of service accounts

### Changed
- [#11] **Breaking Change ahead!** The secret containing the dogu registry data was split and renamed from 
  `dogu-registry-com` to `k8s-dogu-operator-dogu-registry` and `k8s-dogu-operator-docker-registry`.
  It also received the registry endpoint as an additional literal besides username and password. Existing user
  need to delete their old secret and create two new ones. The creation process is described 
  [here](docs/operations/configuring_the_docker_registry_en.md) and
  [here](docs/operations/configuring_the_dogu_registry_en.md).
- [#2] Update makefiles to version 5.0.0

## [v0.2.0] - 2022-04-01
### Added
- [#8] Add the opportunity to process custom dogu descriptors with configmaps
- [#8] Use status field of the dogu resource to identify its state
- [#4] Add functionality to remove dogus
- [#4] Restrict the dogu-operator with rbac resources to operate only in the configured namespace

### Changed
- [#8] Ignore incoming dogu resources if their specs did not change
    - this is likely to happen after status updates where the old and new dogu specs do not differ

## [v0.1.0] - 2022-03-18
### Added
- [#1] initial release
