# Zwischen Staging- und Prod-Registries wechseln

v3-Dogus werden zum Testen zuerst in die Staging-Umgebung released. Um sie zu installieren, muss der Operator beide
Staging-Registries verwenden:

| Zweck                | Secret                     | Prod                                     | Staging                                          |
|----------------------|----------------------------|------------------------------------------|--------------------------------------------------|
| Helm-Charts (Harbor) | `ces-container-registries` | `registry.cloudogu.com`                  | `staging-registry.cloudogu.com`                  |
| Dogu-Registry v3     | `dogu-registry-v3`         | `https://dogu.cloudogu.com/api/v3/dogus` | `https://staging-dogu.cloudogu.com/api/v3/dogus` |

Die Targets arbeiten im Namespace aus `NAMESPACE` und im Kube-Kontext aus `KUBE_CONTEXT_NAME` (standardmäßig der
aktuelle Kontext), beide üblicherweise in der `.env` gesetzt.

## Zugangsdaten

Das Staging-Harbor und das Staging-DCC verwenden dieselben Zugangsdaten. Diese werden in der `.env` hinterlegt:

```bash
STAGING_USERNAME=my-user
STAGING_PASSWORD=my-password
```

Sind sie nicht gesetzt, fragen die Targets sie ab (das Passwort verdeckt). Die `.env` wird von `make` gelesen, ein `$`
im Passwort muss daher als `$$` und ein `#` als `\#` geschrieben werden. Leerzeichen am Ende werden Teil des Werts.

Die Staging-Hosts können über `STAGING_REGISTRY_HOST` und `STAGING_DCC_ENDPOINT` überschrieben werden.

## Wechsel auf Staging

```bash
make registry-staging
```

- Fügt `ces-container-registries` einen Eintrag für `staging-registry.cloudogu.com` hinzu. Alle anderen Einträge
  bleiben unverändert, da v2-Dogus das Secret als `imagePullSecret` nutzen.
- Sichert die Daten des aktuellen `dogu-registry-v3` im Secret `dogu-registry-v3-prod-backup`. Ein bestehendes Backup
  wird nie überschrieben, das Target kann also gefahrlos mehrfach ausgeführt werden. Existiert `dogu-registry-v3`
  nicht, wird es angelegt und das Backup vermerkt, dass es vorher nicht existierte.
- Setzt `endpoint`, `username` und `password` von `dogu-registry-v3` auf Staging. `urlschema` und `insecureSkipVerify`
  bleiben unverändert.
- Startet den Operator neu, da er `dogu-registry-v3` nur beim Start liest.

Das Target schlägt fehl, wenn `ces-container-registries` nicht existiert, da dies meist auf den falschen Cluster oder
Namespace hindeutet.

## Zurück auf Prod

```bash
make registry-prod
```

- Entfernt den Staging-Eintrag aus `ces-container-registries`.
- Stellt die Daten von `dogu-registry-v3` aus `dogu-registry-v3-prod-backup` wieder her und löscht das Backup. Labels
  und Annotationen von `dogu-registry-v3` bleiben unverändert. Existierte das Secret vor dem Wechsel auf Staging nicht,
  wird es gelöscht. Ohne Backup bleibt `dogu-registry-v3` unverändert.
- Startet den Operator neu.

## Aktuellen Stand anzeigen

```bash
make registry-env-status
```

Gibt den Endpoint von `dogu-registry-v3` aus, ob ein Prod-Backup existiert, sowie Host und Benutzernamen jedes Eintrags
in `ces-container-registries`. Passwörter werden nie ausgegeben.

## Hinweise

- Installierte Dogus behalten die Chart-URL ihres `OCIRepository`, bis sie neu installiert oder aktualisiert werden.
  Ein Wechsel der Umgebung wirkt sich nur auf danach installierte oder aktualisierte Dogus aus.
- Werden die Secrets vom CES-Setup neu erzeugt, geht die Staging-Konfiguration verloren. In dem Fall
  `make registry-staging` erneut ausführen.
