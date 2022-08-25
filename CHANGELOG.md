# k8s-dogu-operator Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Changed
- [#38] Detect existing PVC when installing a dogu. This allows users to store initial data for dogus before their installation.
- [#38] Update `ces-build-lib` to version 1.56.0

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
- [#6] Installing generic kubernetes resources when installing a dogu. These resources need to be provided by the dogu image
at the `k8s` folder in the root path (`/k8s`):
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
