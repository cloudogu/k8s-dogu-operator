# k8s-dogu-operator Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.2.0] - 2022-04-01
### Added
- Add the opportunity to process custom dogu descriptors with configmaps #8
- Use status field of the dogu resource to identify its state #8
- Add functionality to remove dogus #4
- Restrict the dogu-operator with rbac resources to operate only in the configured namespace #4
- [#2] Annotation `k8s-dogu-operator.cloudogu.com/ces-services` to Dogu-`Services` containing information of
related CES services. For more information see [Annotations](/docs/operations/annotations_en.md).
  
### Changed
- [#2] Update makefiles to version 5.0.0

### Changed
- Ignore incoming dogu resources if their specs did not change #8
    - this is likely to happen after status updates where the old and new dogu specs do not differ


## [v0.1.0] - 2022-03-18
### Added
- initial release #1
