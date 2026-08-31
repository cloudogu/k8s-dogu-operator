# Das `v3beta1`-Conversion-Webhook

Dieses Dokument beschreibt die `v3beta1`-API-Version der Dogu-CRD, den Conversion-Webhook, den sie
unterstützt, die "doguv2-Guards", die die Geschäftslogik des Operators davor schützen, sowie die
Helm-Ressourcen, die der Webhook benötigt.

## Hintergrund: eine zweite ausgelieferte API-Version

`k8s-dogu-lib/v2` definiert zwei API-Versionen für die Dogu-CRD: `v2` und `v3beta1`. Beide werden
ausgeliefert ("served"), aber `v2` ist die **Storage-Version** (`+kubebuilder:storageversion` steht bei
`v2.Dogu`, nicht bei `v3beta1.Dogu`) — siehe ["Warum `v2` die Storage-Version
ist"](#warum-v2-die-storage-version-ist) weiter unten für die Begründung.

`v3beta1.Dogu` ist der Conversion-**Hub** (`func (*Dogu) Hub() {}`) — der kanonische Typ, über den die
Conversion-Maschinerie konvertiert —, und `v2.Dogu` implementiert `conversion.Convertible` über
`ConvertTo`/`ConvertFrom`:

- `ConvertTo` (v2 → v3beta1, läuft immer dann, wenn ein `v2`-Objekt als `v3beta1` dargestellt werden
  muss, z. B. wenn ein `v3beta1`-Client ein gespeichertes `v2`-Objekt liest) teilt `Spec.Name`
  (`"namespace/name"`) in `DoguNamespace`/`Name` auf, bildet Resources/Security/Mounts ab und setzt
  `Spec.DoguApiVersion` standardmäßig auf `v2`. Felder, die es nur in `v3beta1` gibt
  (`DoguApiVersion`, `Values`), werden über Transport-Annotationen
  (`k8s.cloudogu.com/v3beta1-doguApiVersion`, `k8s.cloudogu.com/v3beta1-values`,
  `k8s.cloudogu.com/v3beta1-status-appVersion`) durchgereicht, die hier konsumiert und wieder entfernt
  werden.
- `ConvertFrom` (v3beta1 → v2, läuft immer dann, wenn ein `v3beta1`-Objekt als `v2` gespeichert/
  ausgeliefert werden muss, z. B. wenn ein `v2`-Client liest oder der Apiserver einen `v3beta1`-Write
  persistiert) macht die umgekehrte Abbildung und legt `DoguApiVersion`/`Values`/`Status.AppVersion` in
  denselben Annotationen ab, außer sie entsprechen bereits dem v2-Standardwert (damit ein
  tatsächlich-v2-Dogu ohne Annotations-Ballast round-tripped).
- `v2.Dogu.IsV2()` liefert `true`, wenn keine `doguApiVersionAnnotationKey`-Annotation vorhanden ist
  oder sie `"v2"` ist; sonst `false`. So kann ein `v2`-typisiertes Objekt erkennen, ob es eigentlich von
  einem Nicht-v2-Dogu stammt.

Weil zwei ausgelieferte Versionen mit nicht-trivialem Schema-Unterschied nebeneinander existieren,
benötigt der Kubernetes-Apiserver für die Dogu-CRD ein **Conversion-Webhook**, unabhängig davon, welche
Version die Storage-Version ist. Der Operator implementiert und stellt dieses Webhook bereit, wie unter
["Verdrahtung des Webhook-Servers"](#verdrahtung-des-webhook-servers) beschrieben.

## Warum `v2` die Storage-Version ist

`v2` als Storage-Version zu behalten bedeutet: Solange ein Client eine Dogu-Resource als `v2` liest oder
schreibt, liefert der Apiserver sie direkt aus dem etcd aus — **ohne Aufruf des Conversion-Webhooks**. Der
Webhook greift nur im Minderheitsfall — wenn ein Client explizit `v3beta1` verwendet. Wäre stattdessen
`v3beta1` die Storage-Version, wäre es umgekehrt: Jedes `v2`-Lesen/-Schreiben — der weit überwiegende
Teil des Traffics — bräuchte einen Conversion-Webhook-Roundtrip.

Die Migration von Ökosystemen von `v2` zu `v3beta1` ist ein langsamer, schleichender Prozess — viele
Systeme nutzen `v2`-Dogus noch sehr lange weiter. `v3beta1` zur Storage-Version zu machen, würde einen
kleinen Anteil an `v3beta1`-Traffic gegen einen Webhook-Aufruf bei jedem `v2`-Reconcile eintauschen —
clusterweit, solange diese Migration andauert. Deshalb bleibt `v2` die Storage-Version, bis die Nutzung
von `v3beta1` verbreitet genug ist, um den Wechsel zu rechtfertigen.

Der Webhook selbst entfällt dadurch nicht — er bleibt erforderlich und ist weiterhin wie unten
beschrieben verdrahtet — er liegt nur nicht auf dem häufig durchlaufenen Pfad des Normalfalls.

## Verdrahtung des Webhook-Servers

- `controllers/initfx/controllerManager.go` stellt `GetWebhookServer()` bereit, eine per fx
  injizierbare Factory, die `webhook.NewServer(webhook.Options{Port: 9443})` zurückgibt. `main.go` fügt
  `initfx.GetWebhookServer` den fx-Options hinzu; `NewManagerOptions(args, operatorConfig, webhookServer
  webhook.Server)` erhält sie und weist sie `manager.Options.WebhookServer` zu.
- Die Scheme-Registrierung im `init()` derselben Datei ruft `doguscheme.AddToScheme(scheme)` auf (Paket
  `k8s-dogu-lib/v2/client/scheme`), was **sowohl** `v2` als auch `v3beta1` registriert — beide
  API-Versionen müssen dem Scheme bekannt sein, damit Conversion funktioniert.
- `controllers/dogu_controller.go` definiert `webhookRegister = func(mgr ctrlManager) error { return
  (&v3beta1.Dogu{}).SetupWebhookWithManager(mgr) }` als paketweite Variable statt als einfachen
  Funktionsaufruf, damit sie in Tests ausgetauscht werden kann. `DoguReconciler.setupWithManager(mgr)`
  ruft `webhookRegister(mgr)` auf, bevor der Controller aufgebaut wird. `SetupWebhookWithManager` wird
  von der Dogu-Lib generiert und registriert den eigentlichen `/convert`-Handler am Webhook-Server des
  Managers.
- `addChecks(mgr)` in `controllers/initfx/controllerManager.go` registriert
  `mgr.AddReadyzCheck("webhook-server", mgr.GetWebhookServer().StartedChecker())`. Damit hängt die
  Pod-Readiness daran, dass das TLS-Zertifikat des Webhooks geladen ist (eine im Code vermerkte mögliche
  künftige Verbesserung wäre zusätzlich, das `caBundle` der CRD mit `ca.crt` zu vergleichen).
- Das `WebhookServer`-Interface (`type WebhookServer interface { webhook.Server }`) in
  `controllers/interfaces.go` existiert einzig, damit mockery `MockWebhookServer`
  (`controllers/mock_WebhookServer_test.go`, `controllers/initfx/mock_WebhookServer_test.go`) generieren
  kann. Tests verwenden dieses Fake, statt einen echten HTTPS-Listener aufzusetzen; siehe
  `TestDoguReconciler_webhookRegister` in `controllers/dogu_controller_test.go`.

## Doguv2-Guards

Weil der Conversion-Webhook ermöglicht, dass `v3beta1`-native Dogus im Cluster existieren, die
Geschäftslogik des Operators aber nur `v2`-Semantik versteht, schützen sich zwei Reconciler davor,
etwas zu verarbeiten, das nicht v2 ist:

- `DoguReconciler.Reconcile` (`controllers/dogu_controller.go`) prüft `doguResource.IsV2()` direkt nach
  dem Laden der Resource. Ist sie nicht v2, wird ein Fehler geloggt ("the operator currently only
  supports v2 dogus.") und `ctrl.Result{}, nil` zurückgegeben — das Reconcile wird stillschweigend
  verworfen, ohne Requeue, ohne Event.
- `DoguRestartReconciler.createRestartInstruction` (`controllers/dogurestart_controller.go`) führt die
  gleiche Prüfung durch, gibt sie aber als Fehler über den Restart-Instruction-/Error-Pfad
  (`handleGetDoguRestartFailed`) weiter.

Das sind Übergangslösungen, keine v3-Implementierung: Sie existieren, damit der Operator sich bei einer
Dogu-Variante, die er nicht unterstützt, sicher verhält, statt sich falsch zu verhalten. Testabdeckung
in `controllers/dogu_controller_test.go`, `controllers/dogu_requeue_handler_test.go` und
`controllers/dogurestart_controller_test.go` baut Dogus mit der Annotation
`k8s.cloudogu.com/v3beta1-doguApiVersion: v3beta1` auf, um diesen Guard zu prüfen.

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
Reconcile aus. Der Kommentar im Code weist vorausschauend darauf hin: *Würde* die Storage-Version
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
ist (sie kann bei verwalteten Control-Planes außerhalb des Pod-CIDRs des Clusters liegen).

### `deployment.yaml`

Der Manager-Container legt `containerPort: 9443, name: webhook-server` offen und mountet ein
`webhook-cert`-Volume (ein `secret`-Volume, `secretName: k8s-dogu-operator-webhook-cert`,
`optional: false`) unter `/tmp/k8s-webhook-server/serving-certs` — dem Standardpfad von
controller-runtime für Webhook-Zertifikate.
