# Additional dogu mounts

The attribute `spec.addionalMounts` can be used to mount files in Dogus.
A description of the format can be found in the repo of the
[Dogu-CRD](https://github.com/cloudogu/k8s-dogu-lib/docs/operations/dogu_format_en.md##AdditionalMounts).

The use of additionalMounts requires that the affected Dogu has a `localConfig` volume.
This is used so that the init container can save mounted files and thus later delete them from the Dogu volumes again later.

The repository for the init container can be found [here](https://github.com/cloudogu/dogu-data-seeder).

The Dogu operator uses the `Name` attribute of the DataMount as the name for the volume.
The [naming guidelines](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#rfc-1035-label-names) therefore apply to volume names.
