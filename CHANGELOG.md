# k8s-dogu-operator Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v3.11.2] - 2025-07-09
### Fixed
- [#253] Only update deployment for export mode when needed
- [#253] Sort capabilities in securityContext to prevent changes in the pod-template due to a different order of the capabilities

## [v3.11.1] - 2025-07-07
### Fixed
- [#148] Fix metadata file to use correct key

## [v3.11.0] - 2025-07-04
### Changed
- [#255] updated exporter-sidecar to registry.cloudogu.com/k8s/rsync-sidecar:1.1.0

### Added
- [#255] dogu-name as environment-variable for exporter-sidecar-container

### Fixed
- [#251] Use dogu name as default pod for command execution. 
  - enable service account creation while export mode is active

## [v3.10.0] - 2025-07-02
### Added
- [#148] Metadata Mapping for log level

## [v3.9.0] - 2025-06-25
### Added
- [#247] Implemented status field `dataVolumeSize` and the condition `meetsMinVolumeSize`. These fields will be updated on the operator start and every volume provisioning.

## [v3.8.2] - 2025-06-12
### Fixed
- [#248] Use correct user and group ids for the `additionalMounts` Init-Container.
  - Note that UIDs/GIDs need to be uniform for all dogu volumes, otherwise this feature may not work as expected
- [#248] update to image cloudogu/dogu-additional-mounts-init:0.1.2, so that user and group ids are set correctly.

## [v3.8.1] - 2025-06-12

- Release fehlgeschlagen

## [v3.8.0] - 2025-06-05
### Added
- [#240] Init-Container creation for `additionalMounts` from the Dogu-CRD.
  The dogu-operator now supports mounting configmaps or secrets in dogus with the `dogu-additional-mounts-init` container.
  - If only the `additionalMounts` change and there is no dogu upgrade, the dogu.json used for this process is fetched from the local dogu registry.
  - Thus, you can add a new volume while developing a dogu and test the `addtionalMounts`.
  - On dogu upgrades the routine will always use the new dogu descriptor from remote.

## [v3.7.0] - 2025-06-04
### Fixed
- [#245] Check Replica count instead of readiness status at Dogu startup to prevent blocking operator when a Dogu cannot be started

## [v3.6.0] - 2025-05-22
### Changed
- [#239] Extracted Dogu-CRD, Dogurestart-CRD and associated clients to own [repository](https://github.com/cloudogu/k8s-dogu-lib)
- [#242] Prevent decreasing volume size
- [#242] Update to dogu resource with new minimal volume size field

## [v3.5.1] - 2025-05-02
### Changed
- [#236] Set sensible resource requests and limits
- [#237] Set resource requests and limits in all containers of dogu, including init container

### Fixed
- A bug where the operator would not register the new dogu version on upgrade.

## [v3.5.0] - 2025-04-02
### Fixed
- [#210] A bug where the operator tried to create a service account for example CAS when it was not healthy.
This occurred in situations where the producer dogu got upgraded and immediately after that an installation with a service account create for that dogu happened.

## [v3.4.0] - 2025-03-31
### Added
- [#234] Add additional print columns and aliases to CRDs

## [v3.3.0] - 2025-03-24
## Added
- [#231] Export-Mode on Dogu-CR
  - This change adds an additional exporter-sidecar container to the pod of the dogu if the exportMode of a dogu is active

## Changed
- [#231] Update to go v1.24.1
- [#231] Update dogu operator CRD to 2.5.0

## [v3.2.1] - 2025-01-28
## Removed
- [#227] Remove allowPrivilegeEscalation flag

## [v3.2.0] - 2025-01-27
### Added
- [#225] Proxy support for the container and dogu registry. The proxy will be used from the secret `ces-proxy` which will be created by the setup or the blueprint controller.
- [#222] Functionality to set security-specific fields in Dogu descriptors and CRs.
  - These will be used to generate a security context for the deployment.

## [v3.1.1] - 2024-12-19
### Fixed 
- [#223] Removed unnecessary rbac proxy to fix CVE-2024-45337

## [v3.1.0] - 2024-12-16
### Added
- [#218] Missing RBACs for events
- [#216] Annotation for exposed ports

### Removed
- [#218] Leader-election. It is not necessary as we do not scale for now.
- [#216] Exposing services

### Fixed
- [#218] Problem with missing RBACs for events

## [v3.0.3] - 2024-12-12
### Added
- [#215] Create network policies for all dogus and their component-dependencies
- [#208] Disable default service-account auto-mounting for dogus
- [#208] Disable service-account token auto-mounting for exec-pods

## [v3.0.2] - 2024-12-05
### Added
- [#212] NetworkPolicy to deny all ingress traffic to this operator
 
### Changed
- [#204] fetch dogu descriptors with retry

### Added
- [#211] Create network policies for all dogus and their dogu-dependencies

## [v3.0.1] - 2024-10-29
### Fixed
- [#205] Use correct apiVersion `v1` in component patch template.

## [v3.0.0] - 2024-10-28
### Changed
- [#201] **Breaking**: The name of secret containing the container registry configurations changed from `k8s-dogu-operator-docker-registry` to `ces-container-registries`.
Use this secret and instead of mounting this as an environment variable the dogu-operator mount it as a file `/tmp/.docker/config.json`.
Add the environment variable `DOCKER_CONFIG` so that crane can use the configuration as default.

## [v2.3.0] - 2024-10-24
### Changed
- [#200] Restrict RBAC permissions as much as possible

## [v2.2.1] - 2024-10-18
### Changed
- [#198] Change go module to v2
- [#198] Change dogu api to v2
- [#198] Change mocks to be inpackage and testonly
- [#198] Change go version to 1.23.2
- [#198] Change makefile version to 9.3.1

## [v2.2.0] - 2024-09-25
### Changed
- [#196] Update k8s-registry-lib to v0.4.1

### Fixed
- [#192] Add missing clientSet-dependency to ManagerSet
    - This fixes a bug when removing component service-accounts
- [#190] Fix a bug where the dogu operator could not install dogus with optional dependencies because the old etcd not found error was used in dependency validation instead of the k8s not found error.

## [v2.1.0] - 2024-09-18
### Changed
- Relicense to AGPL-3.0-only

## [v2.0.1] - 2024-08-08
### Fixed
- [#187] Fix dependency for k8s-dogu-operator-crd in helm-chart
  - Now depends on `k8s-dogu-operator-crd:2.x.x-0` 

## [v2.0.0] - 2024-08-08
**Breaking Change ahead**
### Removed
- [#184] Remove support for internal ETCD

### Changed
- [#184] Add k8s-registry lib in version 0.2.2 to use config maps for configuration instead of the etcd.
  - This change requires all other installed dogus to use doguctl >= v0.12.1

## [v1.2.0] - 2024-06-12
### Added
- [#181] Handle dogu health states with a config map and provide dogus the volume mounts

### Changed
- [#182] Update dogu upgrade docs not to use doguctl state for handling upgrades

## [v1.1.0] - 2024-05-29
### Fixed
- [#171] Fix unnecessary creation of dogu PVCs.
- [#173] Fix start dogu-operator if dogu-cr is in cluster without a deployment

### Changed
- [#171] Only create PVCs for dogus with volumes that need backup.
- Update go version to 1.22
- Update go dependencies
- [#174] Use ConfigMaps in parallel to ETCD for the local dogu registry
- [#176] Add environment variable `ECOSYSTEM_MULTINODE` to identify if dogu is running in multinode.
- [#179] Use local dogu registry from k8s-registry-lib

## [v1.0.1] - 2024-03-22
### Fixed
- [#169] Fix dogu-operator-crd dependency version.

## [v1.0.0] - 2024-03-21

### Attention 
- This release is broken due to an invalid helm dependency version for the `dogu-operator-crd`

### Added
- [#149] Clarified escaping rules for running the operator locally
  (see [here](docs/development/development_guide_en.md) or [here](.env.template))
- [#151] Add field `stopped` in Dogu to start or stop the Dogu.
- [#151] Add new CRD `DoguRestart` to trigger a dogu restart.
  - The reconciler uses the `stopped` field from the Dogu.
- [#159] Manage Service Accounts provided by components
- [#162] Add start and shutdown handler to refresh the dogu health states.
- [#158] Add installed version to dogus status to be able to check the exact state of the dogu.

### Changed
- [#154] Only accept dogu volume sizes in binary format.
- [#156] Stabilized process when updating the status of the dogu cr.

### Fixed
- [#152] The health routine no longer marks a dogu as available if the deployment was scaled to 0.
- [#153] Fix dogu status of restart routine.
- [#167] Select dogu restart resources pro dogu for garbage collection.

## [v0.41.0] - 2024-01-23
### Changed
- Update go dependencies
- Particularly update the k8s-libraries to v0.29.1
- [#141] Improve documentation for running an offline Cloudogu EcoSystem.

## [v0.40.0] - 2024-01-03
### Added
- [#143] Track health on dogu CR

## [v0.39.2] - 2023-12-19
### Fixed
- [#145] Dogu startupProbe timeouts in airgapped environments
### Added
- [#145] Configurable startupProbe timeout

## [v0.39.1] - 2023-12-12
### Fixed
- [#139] Fix missing value for attribute `chownInitImage` in patch templates.

## [v0.39.0] - 2023-12-08
### Added
- [#137] Patch-template for mirroring this operator and its images
### Changed
- [#135] Replace monolithic K8s resource YAML into Helm templates
- Update Makefiles to 9.0.1

## [v0.38.0] - 2023-10-05
### Added
- [#133] Add CRD-Release to Jenkinsfile

### Changed
- [#130] updated go dependencies

### Fixed
- [#130] deprecation warning for argument `logtostderr` in kube-rbac-proxy

### Removed
- [#130] deprecated argument `logtostderr` from kube-rbac-proxy

## [v0.37.0] - 2023-09-15
### Changed
- [#128] Move component-dependencies to helm-annotations

### Removed
- this release cleans up unused code parts that are no longer required: no functionality has been changed

## [v0.36.0] - 2023-09-07
### Added
- [#118] Make implicitly used init container images explicit and configurable
   - this release adds a mandatory ConfigMap `k8s-dogu-operator-additional-images` which contains additionally used images
   - see the [operations docs](docs/operations/installing_operator_into_cluster_en.md) for more information
- [#125] Validate that `metadata.Name` equals simple dogu name in `spec.Name`.

### Fixed
- [#121] Operator cannot recognize multiple changes/required operations at once.
  - Now multiple required operations are detected and after the first operation is done, a requeue is triggered to execute the other ones.
- [#117] Fix waiting for PVC to be resized on "AzureDisk"-storage
  - The conditions "FileSystemResizePending" has to be checked for storage-interfaces (like "AzureDisk") that require a file system expansion before the additional space of an expanded volume is usable by pods.

## [v0.35.1] - 2023-08-31
### Added
- [#119] Add "k8s-etcd" as a dependency to the helm-chart

## [v0.35.0] - 2023-08-14
### Changed
- [#113] Prevent nginx HTTP 413 errors for too small body sizes in SCM-Manager and Jenkins in dogu resource samples
   - A default value of 1 GB per request is now in place
- Update versions for SCM-Manager (2.45.1-1) and Jenkins (2.401.3-1) in the sample dogu resources

### Fixed
- [#115] Fixes conflicts on status update during dogu installation 

## [v0.34.0] - 2023-07-07
### Added
- [#111] Add Helm chart release process to project

### Changed
- [#109] Dogu-volumes without backup (needsBackup: false) are now mounted to an emptyDir-volume.
  Dogu-volumes with backup (needsBackup: true) are mounted to the Dogu-PVC.

## [v0.33.0] - 2023-06-23
### Changed
- [#106] Resource limits (memory, cpu-cores, ephemeral storage) are now read from
  `/config/<dogu>/container_config/<resource-type>_limit` instead of `/config/<dogu>/pod_limit/<resource-type>`.
- [#106] Resource request are now handled separately from limits and can be configured through `/config/<dogu>/container_config/<resource-type>_request`.
- [#106] Defaults for these requests and limits can now be set in the `Configuration`-section of the `dogu.json`.
  These will be used if the key is not configured in the config registry.

### Fixed
- [#108] Failing execs on pods because of missing `VersionedParams`

## [v0.32.0] - 2023-06-21
### Changed
- [#104] Change the pre-upgrade process, so that it doesn't need to create the additional reserved volumes anymore. 
  To do so, we adapted the way of the k8s api (`kubectl cp`) and copied the script directly in the old container 
  by using `tar`. 

## [v0.31.0] - 2023-06-05
### Changed
- [#102] Generate only one loadbalancer service for all dogu exposed ports so that all will be available with the same
  ip. `Nginx ingress` needs additional information to route tcp and udp traffic. The dogu operator creates and updates
  configmaps (`tcp-services` and `udp-services`) for that.

## [v0.30.0] - 2023-05-12
### Added
- [#98] Support for service rewrite mechanism

### Removed
- [#100] Longhorn validation for PVCs

## [v0.29.2] - 2023-04-14
### Fixed
- [#96] Trim "dogus/" suffix only on URL "default" schema
  - this change avoids removing the endpoint suffix for the "index" schema

## [v0.29.1] - 2023-04-11
### Fixed
- [#93] Delete additional ingress annotations if not present on the dogu resource
- [#94] Correct ingress annotation in docs and sample

## [v0.29.0] - 2023-04-06
### Added
- [#91] Add additional ingress annotations to dogu resource. Append those annotations to the dogu's service.

## [v0.28.0] - 2023-04-04
### Changed
- [#89] Add retry mechanism when pulling image metadata to avoid installation/upgrade interrupts if errors occur.  
Moreover, increase the backoff time to 10 minutes when waiting for an exec pod to pull the dogu image.

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
  [here](docs/operations/configuring_the_container_registry_en.md) and
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
