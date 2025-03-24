# Guide for Releasing the CRD Helm Chart

This guide is when the CRD has changed must be released.

> [!IMPORTANT] Release the CRD Helm chart first, the operator Helm chart second
> 
> This way dependency errors from operator to CRD can be caught earlier and the operator uses the correct version of the CRD.

> [!IMPORTANT] Major CRD version changes may break the operator.
>
> Double check if a major version change is really the thing you want and also check the version dependency annotation `k8s.cloudogu.com/ces-dependency/k8s-dogu-operator-crd` in `$workspace/k8s/helm/Chart.yaml`

1. change to the `develop` branch, pull in any further changes
2. ensure the new feature/bugfix arrived in `develop` 
3. log-in into the desired Helm registry
   - `helm registry login ${target-registry}`
   - keep your account log-in and passphrase handy
4. find the current CRD component version
   - look into the desired Helm registry
5. decide on the new version (here `${NewCRDVersion}`)
6. create the CRD Helm chart
   - `DEV_CRD_VERSION=${NewCRDVersion} make crd-helm-package`
   - the version is only defined in the `target/k8s/helm-crd/Chart.yaml` file
   - a file with the corresponding version should reside under `target/k8s/helm-crd/k8s-dogu-operator-crd-${NewCRDVersion}.tgz` 
7. review and commit any changes to YAML files that might occur during `kubebuilder` usage
   - these should consist only of changes to blanks or the kubebuilder version
8. push the new Helm chart version to the registry
   - `helm push target/k8s/helm-crd/k8s-dogu-operator-crd-${NewCRDVersion}.tgz oci://${target-registry}/k8s/`
9. release the operator Helm chart as usual
   1. run `make controller-release`
   2. keep in mind, the operator version may have a totally different version number, though
   3. while reviewing `CHANGELOG.md`, mention the CRD release of `${NewCRDVersion}`
   4. continue to finish the release
