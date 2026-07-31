# Anleitung zum Release des CRD-Helm-Charts

Diese Anleitung gilt, wenn sich die CRD geändert hat und veröffentlicht werden muss.

> [!IMPORTANT] Veröffentliche zuerst das CRD-Helm-Chart und danach das Operator-Helm-Chart.
>
> Dadurch können Abhängigkeitsfehler zwischen Operator und CRD früher erkannt werden, und der Operator verwendet die korrekte Version der CRD.

> [!IMPORTANT] Major-Versionsänderungen der CRD können den Operator inkompatibel machen.
>
> Prüfe genau, ob eine Major-Versionsänderung wirklich gewünscht ist, und prüfe außerdem die Versionsabhängigkeitsannotation `k8s.cloudogu.com/ces-dependency/k8s-dogu-operator-crd` in `$workspace/k8s/helm/Chart.yaml`.

1. wechsle auf den Branch `develop` und ziehe alle weiteren Änderungen
2. stelle sicher, dass das neue Feature oder der Bugfix in `develop` angekommen ist
3. melde dich bei der gewünschten Helm-Registry an
   - `helm registry login ${target-registry}`
   - halte deine Zugangsdaten und Passphrase bereit
4. ermittle die aktuelle CRD-Komponentenversion
   - schaue in der gewünschten Helm-Registry nach
5. entscheide dich für die neue Version (hier `${NewCRDVersion}`)
6. erstelle das CRD-Helm-Chart
   - `DEV_CRD_VERSION=${NewCRDVersion} make crd-helm-package`
   - die Version ist nur in der Datei `target/k8s/helm-crd/Chart.yaml` definiert
   - eine Datei mit der entsprechenden Version sollte unter `target/k8s/helm-crd/k8s-dogu-operator-crd-${NewCRDVersion}.tgz` liegen
7. prüfe und committe alle Änderungen an YAML-Dateien, die während der Verwendung von `kubebuilder` entstehen können
   - diese sollten nur aus Änderungen an Leerzeichen oder der Kubebuilder-Version bestehen
8. pushe die neue Helm-Chart-Version in die Registry
   - `helm push target/k8s/helm-crd/k8s-dogu-operator-crd-${NewCRDVersion}.tgz oci://${target-registry}/k8s/`
9. veröffentliche das Operator-Helm-Chart wie gewohnt
   1. führe `make controller-release` aus
   2. beachte, dass die Operator-Version eine völlig andere Versionsnummer haben kann
   3. erwähne beim Prüfen der `CHANGELOG.md` den CRD-Release von `${NewCRDVersion}`
   4. fahre fort, um den Release abzuschließen
