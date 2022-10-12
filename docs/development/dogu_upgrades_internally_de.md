
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
