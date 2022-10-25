# Dogu-Upgrades

Ein Dogu-Upgrade verläuft in folgenden Schritten:

1. Das Image von DoguV2 wird gepullt.
2. Das Pre-Upgrade-Skript von DoguV2 wird in DoguV1 kopiert und ausgeführt
3. DoguV1 wird heruntergefahren
4. DoguV2 wird hochgefahren und wartet zunächst mit eigentlichem Start
5. Das Post-Upgrade-Skript von DoguV2 wird ausgeführt
6. DoguV2 setzt seine Startroutine fort

## Pre-Upgrade

Im Gegensatz zum herkömmlichen CES ist es nicht so einfach möglich von einem Image Dateien in einen laufenden Container
zu kopieren und dort auszuführen. Ad Hoc ein Volume zu mounten würde einen Neustart des Containers verursachen.
Dies gilt zu verhindern, da die eigentliche Anwendung ebenfalls laufen muss. Bei z.B. Dogus wie Easyredmine würde dies
unnötig Zeit in Anspruch nehmen. 

Für das Kopieren der benötigten Skripte werde bei der Installation eines jeden Dogus Volumes von (10Mb) erzeugt.
Diese Volumes werden ebenfalls von einem Pod eingebunden, der bei der Upgrade-Routine folgende Dinge macht:
1. Image mit Sleep infinity starten
2. Kopieren des Pre-Upgrade-Skriptes in das Dogu-Volume

Nach Beendigung dieser Aktion kann der Dogu-Operator einen Exec-Befehl auf dem Dogu ausführen und somit das Skript starten.
Dabei wird die Startup-Probe auf 10 Minuten erhöht um Neustarts bei langen Upgrade-Routinen (z. B. Migrationen) zu verhindern.



## Notizen

/*

/create-service-account.sh ro redmine

command:
/create-service-account.sh
- ro
- redmine

---

command:
cat

/bin/env bash -c "cat << EOF | /bin/env bash
#!/bin/env bash

# a comment

if [[ "" != "asdf" ]]; then
echo foo
else
echo "rm -rf *"
fi

echo "'all is good'"
echo 'huhu'
echo '"huhu2"'
EOF"
*/

/* preferred way: jedes Dogu erhält ein Upgradereservat
[ redmine original                  ]
[ läuft...                          ] | [ redmine sidecar                  ] <- eigenes Deployment ähnl Volume
[ läuft...                          ] | [ create file from configMap as +x ]    nicht mit dogu deployment
[ läuft...                          ] | [ copy file to upgrade-reservation ]
[                                   ] | [ exit                             ]
[ run file in upgrade-reserveration ]
[ continue upgrade                  ]
*/
/* Nachteil: Image muss Tooling unterstützen (schwierig z. B. bei Anbieterdogus)
[ redmine original                   ]
[ Operator cat's file into container ]
[ läuft...                           ] <- we may not have chmod at hand
[ läuft...                           ]
[                                    ]
[ run file nicht möglich?            ]
*/
/* Nachteil: dauerhaft parallele Container verschwenden Ressourcen
[ redmine original                                                         ]
[ läuft...                           [ redmine sidecar                   ] ]
[ läuft...                           [ create file from configMap as +x  ] ]
[ läuft...                           [ copy file to upgrade-reservation  ] ]
[                                    [ exit                              ] ]
[ run file in upgrade-reserveration                                        ]
[ continue upgrade                                                         ]
*/

/* Nachteil: Produziert downtime, manche Dogus benötigen eine sehr lange Startzeit
[ redmine original                                                         ]
...deployment wird geändert zu gunsten eines neuen Volumemounts
...z. B. aus einer ConfigMap
... downtime goes here
... :cricket:
... :cricket:
... downtime still there
... oh look, the pod is ready!
[ run file in upgrade-reserveration                                        ]
[ continue upgrade                                                         ]
*/
