# Service Accounts
The dogu operator is responsible for creating service accounts for dogus and deleting them again when a dogu is deleted.

There are three different types of service accounts that can be requested by Dogus:

## Dogu
These service accounts are provided by other Dogus.
The service accounts are created via the Exposed Commands of the requested Dogus and are executed via an exec command in the container of the Dogus.
Further information can be found in the [Dogu Development Docs](https://github.com/cloudogu/dogu-development-docs/blob/4f64940187e11d5970173548cc3a5b52a9367faf/docs/core/compendium_en.md#type-exposedcommand).

## Ces
This is a special form of service account that is only used for `k8s-ces-control`.
A service account is also created here using a fixed exec command. However, this is a special implementation in the Dogu operator that only works for `k8s-ces-control`.

## Component
These service accounts are provided by components.
The components provide an HTTP API for administration.
