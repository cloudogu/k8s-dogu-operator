# Zwischen Staging- und Prod-Registries wechseln

v3-Dogus werden zum Testen zuerst in die Staging-Umgebung released. Um sie zu installieren, muss der Operator beide
Staging-Registries verwenden:

| Zweck                | Secret                     | Prod                                     | Staging                                          |
|----------------------|----------------------------|------------------------------------------|--------------------------------------------------|
| Helm-Charts (Harbor) | `ces-container-registries` | `registry.cloudogu.com`                  | `staging-registry.cloudogu.com`                  |
| Dogu-Registry v3     | `dogu-registry-v3`         | `https://dogu.cloudogu.com/api/v3/dogus` | `https://staging-dogu.cloudogu.com/api/v3/dogus` |

Die Targets arbeiten im aktuellen kubectl-Kontext, im Namespace, der in der `.env` über `NAMESPACE` gesetzt ist.

## Zugangsdaten

Beide Targets fragen Benutzernamen und Passwort der Umgebung ab, auf die gewechselt wird (das Passwort verdeckt). Die
Zugangsdaten werden für Harbor und das DCC verwendet. Ist einer der Werte leer, wird nichts geändert.

Die Hosts können über `STAGING_REGISTRY_HOST`, `STAGING_DCC_ENDPOINT`, `PROD_REGISTRY_HOST` und `PROD_DCC_ENDPOINT`
überschrieben werden.

## Wechsel auf Staging

```bash
make registry-staging
```

- Schreibt den Eintrag für `staging-registry.cloudogu.com` mit den abgefragten Zugangsdaten in
  `ces-container-registries`. Alle anderen Einträge bleiben unverändert, da v2-Dogus das Secret als `imagePullSecret`
  nutzen.
- Setzt `endpoint`, `username` und `password` von `dogu-registry-v3` auf Staging.
- Startet den Operator neu, da er `dogu-registry-v3` nur beim Start liest.

## Zurück auf Prod

```bash
make registry-prod
```

- Entfernt den Staging-Eintrag aus `ces-container-registries` und schreibt den Eintrag für `registry.cloudogu.com` mit
  den abgefragten Zugangsdaten.
- Setzt `endpoint`, `username` und `password` von `dogu-registry-v3` auf Prod.
- Startet den Operator neu.

## Verhalten beider Targets

- Sie schlagen fehl, wenn `ces-container-registries` nicht existiert, da dies meist auf den falschen Cluster oder
  Namespace hindeutet.
- Ein Registry-Eintrag wird unter dem reinen Host gespeichert. Ein bestehender Eintrag desselben Hosts mit
  `https://`-Schema wird ersetzt.
- Von `dogu-registry-v3` werden nur `endpoint`, `username` und `password` geändert. `urlschema`, `insecureSkipVerify`,
  Labels und Annotationen bleiben unverändert. Existiert das Secret nicht, wird es mit `urlschema=default` und
  `insecureSkipVerify=false` angelegt.

## Aktuellen Stand anzeigen

```bash
make registry-env-status
```

Gibt den Endpoint von `dogu-registry-v3` sowie Host und Benutzernamen jedes Eintrags in `ces-container-registries`
aus. Passwörter werden nie ausgegeben.

## Hinweise

- Installierte Dogus behalten die Chart-URL ihres `OCIRepository`, bis sie neu installiert oder aktualisiert werden.
  Ein Wechsel der Umgebung wirkt sich nur auf danach installierte oder aktualisierte Dogus aus.
- Werden die Secrets vom CES-Setup neu erzeugt, geht die Staging-Konfiguration verloren. In dem Fall
  `make registry-staging` erneut ausführen.
