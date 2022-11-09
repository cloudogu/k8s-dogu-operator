# Dogu operator and custom resource definition `Dogu`.

A controller is a Kubernetes application that is informed about state changes of resources that it listens for. Since the [operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) is used for this, they are also called _operators_.

Such operators often come into play in the context of _Custom Resource Definitions_ (CRD) when Kubernetes is to be extended with custom resource types. The Dogu operator is such an operator, which takes care of the management of Dogus in terms of Kubernetes resources.

The basic idea of the operator is relatively simple. It takes care of successful execution of Dogu in a cluster. The operator specifies the resources to be used based on these information:
- dogu.json
- container image
- CES instance credential
  in order to create all required Kubernetes resources, e.g.:
   - Container
   - Persistent Volume Claim
   - Persistent Volume
   - Service
   - Ingress

Each of the Kubernetes resources must be created by a description (usually in YAML format). Because of the amount of resources and the amount of properties per resource, a Dogu installation quickly becomes tedious and error-prone. The Dogu operator provides useful support here by automatically taking care of resource management in Kubernetes. With a few lines of dogu description, a dogu can be installed like this (see below).

The following graphic shows different tasks during a Dogu installation.

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

## Dogu management

The CRD (Custom Resource) description for Dogus looks something like this:

Example: `ldap.yaml`
```yaml
apiVersion: dogu.cloudogu.com/v1
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

To install the LDAP dogu, a simple call is enough:

```bash
kubectl apply -f ldap.yaml
```

With the introduction of the Dogu CRD, Dogus we can use native Kubernetes resources, for example:

```bash
# lists a single dogu
kubectl get dogu ldap
# lists all installed dogus
kubectl get dogus
# delete a single dogu
kubectl delete dogu ldap
```

## dogu operator vs `cesapp`

In terms of their function, dogu operator and `cesapp` are very comparable because both take care of managing and orchestrating dogus in their respective execution environments.

However, in the long run, the Dogu operator will not reach the size and complexity of `cesapp` because its function is very much related to installing, updating and uninstalling Dogus.

## Kubernetes volumes vs Docker volumes.

With few exceptions, Dogus often define volumes in their `dogu.json` where their state should be persisted. In the previous EcoSystem, this was solved by Docker volumes that the `cesapp` set up and assigned to the container during installation.

In Kubernetes, persistence is more decoupled. A Persistent Volume Claim (PVC) defines the size of the desired volume, which in turn is a persistent volume provisioned by a storage provider.

Unlike a Docker volume, a Kubernetes volume cannot easily resize because it is immutable. In addition, separate processes may be started for each Kubernetes volume, which again consume main memory.

The dogu operator creates a single volume for these reasons. All volumes defined in `dogu.json` are then mounted as subdirectories in the volume.
