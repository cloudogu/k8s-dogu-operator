# Dogu-Operator und Custom Resource Definition `Dogu`

Ein Controller ist eine Kubernetes-Applikation, dass über Zustandsänderungen von Ressourcen informiert wird, auf die dieser lauscht. Da hier das [Operator-Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) zum Tragen kommt, nennt man diese auch _Operator_. 

Solche Operatoren kommen häufig im Zusammenhang mit _Custom Resource Definitionen_ (CRD) zum Zuge, wenn Kubernetes um eigene Ressourcentypen erweitert werden soll. Der Dogu-Operator ist solch ein Operator, der sich um die Verwaltung von Dogus im Sinne von Kubernetes-Ressourcen kümmert.

Der Grundgedanke des Operators ist relativ simpel. Er sorgt für eine erfolgreiche Ausführung von Dogu in einem Cluster. Der Operator legt anhand von
- dogu.json
- Container-Image
- CES-Instanz-Credential

alle benötigten Kubernetes-Ressourcen an, z. B.
  - Container
  - Persistent Volume Claim
  - Persistent Volume
  - Service
  - Ingress
  - u. ä.
 
Jede der Kubernetes-Ressourcen muss durch eine Beschreibung (i. d. R. im YAML-Format) angelegt werden. Wegen der Menge der Ressourcen und der Menge an Eigenschaften je Ressource wird eine Dogu-Installation schnell mühselig und fehlerträchtig. Der Dogu-Operator unterstützt hierbei sinnvoll, indem er sich automatisiert um die Ressourcen-Verwaltung in Kubernetes kümmert. Mit wenigen Zeilen Dogu-Beschreibung kann so ein Dogu installiert werden (siehe unten)

Die folgende Grafik zeigt unterschiedliche Aufgaben während einer Dogu-Installation.

```uml
!define CLOUDOGUURL https://raw.githubusercontent.com/cloudogu/plantuml-cloudogu-sprites/master

!includeurl CLOUDOGUURL/common.puml
!includeurl CLOUDOGUURL/dogus/cloudogu.puml
!includeurl CLOUDOGUURL/tools/etcd.puml
!includeurl CLOUDOGUURL/tools/docker.puml
!includeurl CLOUDOGUURL/tools/k8s.puml
!includeurl CLOUDOGUURL/dogus/cas.puml
!includeurl CLOUDOGUURL/dogus/openldap.puml
!includeurl CLOUDOGUURL/dogus/nginx.puml
!includeurl CLOUDOGUURL/dogus/scm.puml
!includeurl CLOUDOGUURL/dogus/redmine.puml
!includeurl CLOUDOGUURL/dogus/postgresql.puml
!define SECONDARY_COLOR #55EE55

rectangle "Cloudogu Backend" as backend <<$cloudogu>> #white {
  TOOL_DOCKER(harbor, "registry.cloudogu.com")
  DOGU_CLOUDOGU(dcc, "dogu.cloudogu.com")
}

rectangle "Cluster" as cluster <<$k8s>> #white {
  TOOL_DOCKER(docker, "Container Runtime"){
    DOGU_REDMINE(redmine, "Redmine") #white
    DOGU_POSTGRESQL(postgresql, "PostgreSQL") #white
    TOOL_ETCD(etcd, "legacy registry") #white
  }

  rectangle "Dogu-Operator" as op <<$k8s>> {
    
  }

  rectangle "Service" as service <<$k8s>>

  rectangle configMap  <<$k8s>> #white {
    file nodeMaster
  }

  rectangle secrets  <<$k8s>> #white {
    rectangle instanceCredentials
  }

  database "volume / volume claim" as vol {
    package subdirectories #white{
      package plugins
      package tmp
    }
  }

  rectangle "Ingress" as ingress
}

op <==u=> dcc : Pull dogu.json (mit Credentials)
op <=u=> harbor : Pull Image (mit Credentials)
op => service : erzeugt Service
instanceCredentials -.- op
op ===> vol : veranlasst Volume-Bereitstellung
op ==> redmine : instanziiert
op ==> postgresql : erzeugt Dogu-Service-Account für Redmine

redmine -.u.-.- nodeMaster : als Volume gemountet
redmine -. postgresql : authentifiziert durch Service Account
redmine <.-d.- vol : als Volume gemountet
redmine <-. service
service <-. ingress
redmine -.-> etcd : benutzt etcd wie gewohnt

cluster -[hidden]u-> backend
harbor ------[hidden]r-> dcc
docker -[hidden]u-> op

actor user

user -u-> ingress : http://server.com/redmine

legend right
Diese Ansicht ignoriert
die Verteilung auf Nodes
endlegend

caption Unterschiedliche Aufgaben des Dogu-Operators bei einer Dogu-Installation
```

## Dogu-Verwaltung

Die CRD-Ausprägung (Custom Resource) für Dogus sieht ungefähr so aus:

Beispiel: `ldap.yaml`

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: ldap
  labels:
    dogu.name: ldap
    app: ces
spec:
  name: official/ldap
  version: 2.4.48-3
```

> [!IMPORTANT]
> `metadata.name` und der einfache Name des Dogus in `spec.name` müssen gleich sein.
> Der einfache Name ist der Teil nach dem Schrägstrich (`/`), also ohne den Namespace.
> Zum Beispiel wäre für eine dogu mit `spec.name` von `k8s/nginx-ingress` der `metadata.name` `nginx-ingress` in Ordnung, während `nginx` nicht in Ordnung wäre.

Um das LDAP-Dogu zu installieren, reicht ein einfacher Aufruf:

```bash
kubectl apply -f ldap.yaml
```

Mit Einführung der Dogu-CRD lassen sich Dogus wir native Kubernetes-Ressourcen verwenden, z. B.:

```bash
# listet ein einzelnes Dogu auf
kubectl get dogu ldap
# listet alle installierten Dogus auf
kubectl get dogus
# löscht ein einzelnes Dogu
kubectl delete dogu ldap
```

## Dogu-Operator vs `cesapp`

Hinsichtlich ihrer Funktion sind Dogu-Operator und `cesapp` sehr vergleichbar, weil beide sich um die Verwaltung und Orchestrierung von Dogus in ihrer jeweiligen Ausführungsumgebung kümmern.

Langfristig wird der Dogu-Operator aber nicht die Größe und Komplexität der `cesapp` erreichen, da seine Funktion sich sehr stark auf die Installation, Aktualisierung und Deinstallation von Dogus bezieht.

## Kubernetes-Volumes vs Docker-Volumes

Mit wenigen Ausnahmen definieren Dogus in ihrer `dogu.json` häufig Volumes, in denen ihr Zustand persistiert werden soll. Im bisherigen EcoSystem wurde dies durch Docker-Volumes gelöst, die die `cesapp` während der Installation eingerichtet und dem Container zugewiesen hat.

In Kubernetes ist die Persistenz stärker entkoppelt. Ein Persistent Volume Claim (PVC) definiert die Größe des gewünschten Volumes, das wiederum ein Persistent Volume durch einen Storage Provider bereitgestellt wird.

Im Gegensatz zu einem Docker-Volume kann ein Kubernetes-Volume nicht ohne weiteres seine Größe ändern, da es immutable ist. Hinzukommt, dass u. U. für jedes Kubernetes-Volume separate Prozesse gestartet werden, die wieder Hauptspeicher verbrauchen.

Der Dogu-Operator legt aus diesen Gründen ein einziges Volume an. Alle in der `dogu.json` definierten Volumes werden dann als Unterverzeichnis in dem Volume hinein gemountet.
