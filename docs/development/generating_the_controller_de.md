# Generierung des dogu-Operators

Dieses Dokument enthält die Anweisung, die zur Erzeugung des Controllers ausgeführt wird. Hierfür wurde `kubebuilder` verwendet.

## Installieren Sie kubebuilder

Für Installationsanweisungen siehe [https://book.kubebuilder.io/quick-start.html#installation](https://book.kubebuilder.io/quick-start.html#installation).

## Controller generieren

Der folgende Befehl wurde im Stammverzeichnis des Controllers ausgeführt.

1. Erzeugen Sie die initiale Struktur des Controllers:

`kubebuilder init --domain cloudogu.com --repo github.com/cloudogu/k8s-dogu-operator`

2. Erzeugen Sie die Api und CRD für den Dogus

`kubebuilder create api --group k8s --version v1 --kind Dogu`