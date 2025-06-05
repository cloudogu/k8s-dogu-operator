# Additional dogu mounts

Mit dem Attribut `spec.addionalMounts` können Files in Dogus gemounted werden.
Ein Beschreibung des Formats ist in dem Repo der 
[Dogu-CRD](https://github.com/cloudogu/k8s-dogu-lib/blob/develop/docs/operations/dogu_format_de.md#additionalmounts) zu finden.

Die Anwendung von additionalMounts setzt voraus, dass das betroffene Dogu ein `localConfig` Volume besitzt.
Dieses wird verwendet, damit der Init-Container gemountete Files speichern und somit später wieder aus den Dogu-Volumes
löschen kann.

Das Repository für den Init-Container ist [hier](https://github.com/cloudogu/dogu-additional-mounts-init) zu finden.

Der Dogu-Operator verwendet das `Name`-Attribute des DataMounts als Namen für das Volume.
Daher gelten die [Namensrichtlinien](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#rfc-1035-label-names) für Volumenamen.

## Beispiel

### Nginx Custom HTML

Eigene HTML-Files können leicht eingebunden werden:

- Dateien im Cluster erstellen

`kubectl create cm myhtml -n ecosystem --from-file=barrierefreiheitserklaerung.html=/files/barrierefreiheitserklaerung.html --from-file=about.html=/files/about.html`

- Mounten der Dateien

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
    - sourceType: ConfigMap # Typ der Quelle [ConfigMap|Secret]
      name: myhtml # Name der ConfigMap
      volume: customhtml # Name des Volumes aus der dogu.json
```

- Dabei könnte ebenfalls ein Subfolder verwendet werden, wenn die Dateien nicht in den Root des Dogu-Volumes kopiert werden sollen:

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
    - sourceType: ConfigMap # Typ der Quelle [ConfigMap|Secret]
      name: myhtml # Name der ConfigMap
      volume: customhtml # Name des Volumes aus der dogu.json
      subfolder: my/page # Subfolder im Zielvolume
```

> Man kann außerdem mehrere Quellen in ein Ziel mounten. Kollidierende Dateinamen werden dabei von der letzten Datei überschrieben.

> Weitere aktuelle Anwendungsfälle zum Mounten von Dateien betreffen folgende Dogus: Sonar (Rules), Teamscale (Analyseprofile), Jenkins (Custom Groovy-Skripte)
