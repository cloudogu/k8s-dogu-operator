package resource

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewResourceManager(client client.Client) *resourceManager {
	return &resourceManager{client: client}
}

type resourceManager struct {
	client client.Client
}

// 1. dogu status auf upgrading stellen

// 2. pre upgrade skript punkt markieren

// 3. (custom) dogu config map für lokale Entwicklung unterstützen
// -- ggf mit custom dogu config map überschreiben
// siehe getDoguDescriptor()

// 4. registriere neue Dogu-Version im etcd
// - TODO CesDoguRegistrator so Umschreiben, dass keine neuen KeyPairs generiert werden

// 5. serviceAccounts ggf installieren
// was passiert mit Service Accounts, die wegfallen => PostUpgrade auch wegen Datenmigration

// 6. image pullen
// ggf. Upgrade-Skripte extrahieren
// 7. Custom K8s Resourcen extrahieren

// 8. Custom K8s resourcen anwenden
// analog zu installManager / wegen apply-lib sollte dies ohne Konflikte stattfinden

// !? RG soll selbst entscheiden, ob Resourcen neu erzeugt oder existierende wieder verwendet werden sollen
// 9. K8s resource beschreibungen erzeugen
//    1. resourceGenerator.createUpgradeDeployment
//    2. volumes usw überschreiben analog RG
//    3. createOrUpdateServiceService
//    4. createOrUpdateExposedServices

// 10. post upgrade skript punkt markieren

// 11. dogu status auf Installed zurücksetzen
// 12. (custom) dogu config map löschen
