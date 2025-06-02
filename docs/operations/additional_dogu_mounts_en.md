# Additional dogu mounts

The attribute `spec.addionalMounts` can be used to mount files in Dogus.
A description of the format can be found in the repo of the
[Dogu-CRD](https://github.com/cloudogu/k8s-dogu-lib/blob/develop/docs/operations/dogu_format_en.md#additionalmounts).

The use of additionalMounts requires that the affected Dogu has a `localConfig` volume.
This is used so that the init container can save mounted files and thus later delete them from the Dogu volumes again later.

The repository for the init container can be found [here](https://github.com/cloudogu/dogu-additional-mounts-init).

The Dogu operator uses the `Name` attribute of the DataMount as the name for the volume.
The [naming guidelines](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#rfc-1035-label-names) therefore apply to volume names.

## Example

### Nginx Custom HTML

Custom HTML files can be easily integrated:

- Create files in the cluster

`kubectl create cm myhtml -n ecosystem --from-file=barrierefreiheitserklaerung.html=/files/barrierefreiheitserklaerung.html --from-file=about.html=/files/about.html`

- Mount the files

```yaml
apiVersion: k8s.cloudogu.com/v2
kind: Dogu
metadata:
  labels:
    app: ces
    dogu.name: nginx-static
  name: nginx-static
  namespace: ecosystem
spec:
  name: k8s/nginx-static
  version: 1.26.3-2
  additionalMounts:
    - sourceType: ConfigMap # Type of source [ConfigMap|Secret]
      name: myhtml # Name of the ConfigMap
      volume: customhtml # Name of the volume from the dogu.json
```

- A subfolder could also be used if the files are not to be copied to the root of the dogu volume:

```yaml
apiVersion: k8s.cloudogu.com/v2
kind: Dogu
metadata:
  labels:
    app: ces
    dogu.name: nginx-static
  name: nginx-static
  namespace: ecosystem
spec:
  name: k8s/nginx-static
  version: 1.26.3-2
  additionalMounts:
    - sourceType: ConfigMap # Type of source [ConfigMap|Secret]
      name: myhtml # Name of the ConfigMap
      volume: customhtml # Name of the volume from the dogu.json
      subfolder: my/page # Subfolder in the target volume
```

> You can also mount several sources in one target. Conflicting file names will be overwritten by the last file.

> Other current use cases for mounting files concern the following Dogus: Sonar (rules), Teamscale (analysis profiles), Jenkins (custom Groovy scripts)
