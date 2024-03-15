# Service Accounts
Der Dogu-Operator ist dafür verantwortlich Service Accounts für Dogus zu erstellen und wieder zu löschen, wenn ein Dogu wieder gelöscht wird.

Es gibt drei verschiedene Arten von Service Accounts, die von Dogus angefordert werden können:

## Dogu
Diese Service Accounts werden von anderen Dogus bereitgestellt.
Die Erstellung der Service Accounts erfolgt über die Exposed Commands des angeforderten Dogus und wird über einen exec-Befehl im Container des Dogus durchgeführt.
Weitere Information dazu sind in den [Dogu Development Docs](https://github.com/cloudogu/dogu-development-docs/blob/4f64940187e11d5970173548cc3a5b52a9367faf/docs/core/compendium_de.md#type-exposedcommand) zu finden.

## Ces
Dies ist eine spezielle Form der Service Accounts, die nur für `k8s-ces-control` verwendet wird.
Hier wird ebenfalls per fest vorgegebenen exec-Befehl ein Service-Account erstellt. Dies ist jedoch eine spezielle Implementierung im Dogu-Operator, die nur für `k8s-ces-control` funktioniert.

## Component
Diese Service Accounts werden von Komponenten bereitgestellt.
Für die Verwaltung stellen die Komponenten eine HTTP-API bereit.
Das genaue Vorgehen ist im [ADR-0015](https://github.com/cloudogu/k8s-ecosystem-architecture/blob/main/adrs/0015-komponenten-service-accounts.md) beschrieben