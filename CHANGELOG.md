# Baseline Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- Add functionality to remove dogus #4
- Restrict the dogu-operator with rbac resources to operate only in the configured namespace #4
- [#2] Annotation `k8s-dogu-operator.cloudogu.com/ces-services` to Dogu-`Services` containing information of
related ces services. For more information see [Annotations](/docs/operations/annotations_en.md).
  
### Changed
- [#2] Update makefiles to version 5.0.0

## [v0.1.0] - 2022-03-18
### Added
- initial release #1
