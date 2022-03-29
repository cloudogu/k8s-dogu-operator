# Baseline Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- Add the opportunity to process custom dogu descriptors with configmaps #8
- Add an update predicate filter to ignore incoming dogu resources with unchanged specs #8
- Use status field of the dogu resource to identify it's state #8
- Add functionality to remove dogus #4
- Restrict the dogu-operator with rbac resources to operate only in the configured namespace #4

## [v0.1.0] - 2022-03-18
### Added
- initial release #1
