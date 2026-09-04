# Der `v3beta1`-Conversion-Webhook

## Hintergrund: eine zweite ausgelieferte API-Version

`k8s-dogu-lib/v3` definiert zwei API-Versionen für die Dogu-CRD: `v2` und `v3beta1`. Beide werden
ausgeliefert ("served"). `v2` ist die **Storage-Version**

Weil zwei ausgelieferte Versionen mit nicht-trivialem Schema-Unterschied nebeneinander existieren,
benötigt der Kubernetes-Apiserver für die Dogu-CRD ein **Conversion-Webhook**, unabhängig davon, welche
Version die Storage-Version ist. Der Operator implementiert diesen mit dem Kubebuilder-Framework.
Für die Kommunikation des API-Servers mit dem Webhook-Server wird zusätzliche Konfiguration benötigt.

## Helm-Ressourcen

### `cert-manager.yaml`

Ein selbstsignierter `Issuer` (`<name>-selfsigned-issuer`) sowie ein `Certificate` mit dem wörtlichen
Namen `k8s-dogu-operator-webhook-cert` — dieser exakte Name ist ein **Vertrag mit `k8s-dogu-lib`**, er
darf nicht ohne Rücksprache mit der Lib umbenannt werden. `Certificate.spec.secretName` entspricht dem,
und die DNS-Namen sind `<name>-webhook.<namespace>.svc[.cluster.local]`.

Deshalb hat der Operator **cert-manager als Laufzeit-Abhängigkeit**: Es stellt das TLS-Zertifikat aus,
das der Webhook-Server verwendet, und rotiert es. Siehe
[install_cert_manager_en.md](install_cert_manager_en.md) zur Installation in einem Cluster.

### `webhook-service.yaml`

Ein `Service` namens `<name>-webhook` (ebenfalls ein Vertrag mit `k8s-dogu-lib`), Port `443` →
`targetPort: webhook-server` (passend zum Container-Port-Namen in `deployment.yaml`).

Er setzt bewusst `publishNotReadyAddresses: true`: Der Apiserver ruft diesen Service auf, um das
Conversion-Webhook auszuführen, und beim Start des Operators lösen alle Dogu-Resourcen sofort ein
Reconcile aus. *Würde* die Storage-Version
jemals auf `v3beta1` umgestellt, bräuchte jedes dieser Start-Reconciles das Webhook sofort — der
Webhook-Server braucht aber ca. 5–10 Sekunden, um bereit zu sein. Ohne `publishNotReadyAddresses` gäbe
es in dieser Zeit noch keine Service-Endpoints, diese Webhook-Aufrufe würden fehlschlagen, die
Reconciles würden fehlschlagen, und der Pod könnte hängen bleiben, ohne je Ready zu werden (ein
selbstverursachtes Deadlock). Da `v2` die Storage-Version ist, treffen die meisten Start-Reconciles den
Webhook aktuell gar nicht, das Risiko ist also latent statt akut — die Einstellung ist aber trotzdem
gesetzt.

### `deny-all-network-policy.yaml`

Über `.Values.global.networkPolicies.enabled` steuerbar. Verweigert jeglichen Ingress zu den
Operator-Pods außer TCP-Port `9443` (der Webhook-Port). Dieser Port ist bewusst für alle Quellen offen,
weil der Kube-Apiserver das Conversion-Webhook direkt aufruft und dessen Quell-IP nicht vorhersehbar
ist.

### `deployment.yaml`

Der Manager-Container legt `containerPort: 9443, name: webhook-server` offen und mountet ein
`webhook-cert`-Volume (ein `secret`-Volume, `secretName: k8s-dogu-operator-webhook-cert`,
`optional: false`) unter `/tmp/k8s-webhook-server/serving-certs` — dem Standardpfad von
controller-runtime für Webhook-Zertifikate.
