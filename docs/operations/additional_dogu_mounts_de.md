# Additional dogu mounts

Mit dem Attribut `spec.addionalMounts` können Files in Dogus gemounted werden.
Ein Beschreibung des Formats ist in dem Repo der 
[Dogu-CRD](https://github.com/cloudogu/k8s-dogu-lib/docs/operations/dogu_format_de.md##AdditionalMounts) zu finden.

Die Anwendung von additionalMounts setzt voraus, dass das betroffene Dogu ein `localConfig` Volume besitzt.
Dieses wird verwendet, damit der Init-Container gemountete Files speichern und somit später wieder aus den Dogu-Volumes
löschen kann.

Das Repository für den Init-Container ist [hier](https://github.com/cloudogu/dogu-data-seeder) zu finden.