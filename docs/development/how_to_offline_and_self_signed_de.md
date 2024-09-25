# Offline-Verwendung des k8s-CES mit selbst signierten Zertifikaten

Dieses Dokument beschreibt stichwortartig, wie das k8s-Ecosystem mit einer lokalen Docker- und Dogu-Registry aufgesetzt
werden kann. Bei der Verwendung ist zu beachten, dass die verwendete FQDN, Credentials des Ecosystems und die Dogu- bzw. Komponenten-Versionen aktuell gehalten werden.

## Aufsetzen der Dogu- und Dockerregistry
- (Legacy-)Ecosystem aufsetzen mit IP `192.168.56.10` und Nexus-Dogu
- FQDN als insecure registry in docker config hinzufügen `/etc/docker/daemon.json`
- raw(hosted) Repository `mirror` und `k8s` anlegen

### Komponenten spiegeln:

`ces-mirror configuration.yaml`:

```yaml
version: 3

k8s:
  cache-directory: .ces-mirror/cache/k8s
  components:
    k8s/k8s-snapshot-controller:
      - "5.0.1-5"
    k8s/k8s-snapshot-controller-crd:
      - "5.0.1-5"
    k8s/k8s-cert-manager-crd:
      - "1.13.1-2"
    k8s/k8s-cert-manager:
      - "1.13.1-2"
    k8s/k8s-velero:
      - "5.0.2-4"
    k8s/k8s-component-operator:
      - "0.7.0"
    k8s/k8s-component-operator-crd:
      - "0.7.0"
    k8s/k8s-backup-operator-crd:
      - "0.9.0"
    k8s/k8s-dogu-operator:
      - "0.39.1"
    k8s/k8s-dogu-operator-crd:
      - "0.39.1"
    k8s/k8s-loki:
      - "2.9.1-4"
    k8s/k8s-minio:
      - "2023.9.23-5"
    k8s/k8s-promtail:
      - "2.9.1-2"
    k8s/k8s-backup-operator:
      - "0.9.0"
    k8s/k8s-host-change:
      - "0.3.2"
    k8s/k8s-ces-setup:
      - "0.20.1"
    k8s/k8s-ces-control:
      - "0.5.0"
    k8s/k8s-longhorn:
      - "1.5.1-3"
    k8s/k8s-service-discovery:
      - "0.15.0"
  source:
    component-index:
      endpoint: https://registry.cloudogu.com/
      username: TODO
      password: TODO
  target:
    component-index:
      type: remote
      endpoint: https://192.168.56.10
      username: ces-admin
      password: ces-admin
      insecure: true
    registry:
      endpoint: https://192.168.56.10
      username: ces-admin
      password: ces-admin
      insecure: true
dogu:
  cache-directory: .ces-mirror/cache/dogus
  dogus:
    official/ldap:
      - 2.6.2-6
    official/postfix:
      - 3.6.4-6
    k8s/nginx-static:
      - 1.23.1-5
    k8s/nginx-ingress:
      - 1.6.4-4
    official/cas:
      - 6.6.12-1
    official/postgresql:
      - 12.15-2
    official/redmine:
      - 5.0.5-2
  docker:
    endpoint: unix:///var/run/docker.sock
  source:
    auth-backend:
      credentials-store: ces-mirror
      endpoint: https://account.cloudogu.com
      proxy:
        enabled: false
        server: localhost
        port: 3128
    dogu-backend:
      endpoint: https://dogu.cloudogu.com/api/v2/
      credentials-store: ces-mirror
      url-schema: default

  target:
    registry:
      endpoint: 192.168.56.10
      username: ces-admin
      password: ces-admin
      insecure: true
    webserver:
      type: remote
      endpoint: https://192.168.56.10/nexus/repository/mirror
      username: ces-admin
      password: ces-admin
      insecure: true
```

- `go run . sync dogu auth`
- `go run . sync dogu`
- `go run . sync k8s`


## Vorbereitung k8s-Ecosystem

### K3S

-  setup.json in `k8s-ecosystem` auf completed `false` setzen
- `vagrant up`
- Zertifikat von (Legacy-)Ecosystem `etcdctl get config/_global/certificate/server.crt` speichern und nach `k8s-ecosystem/cert.pem` kopieren
- Zertifikat auf Maschinen verteilen (für k3s):
  - `vagrant ssh main`
  - `sudo cp /vagrant/cert.pem /etc/ssl/certs/cert.pem`
  - registries.yaml bearbeiten (siehe unten)
  - `sudo systemctl restart k3s`
  - `vagrant ssh worker-0`
  - `sudo cp /vagrant/cert.pem /etc/ssl/certs/cert.pem`
  - registries.yaml bearbeiten (siehe unten)
  - `sudo systemctl restart k3s-agent`

`/etc/rancher/k3s/registries.yaml`:

```yaml
configs:
  "192.168.56.10":
    auth:
      username: ces-admin
      password: ces-admin
    tls:
      ca_file: /etc/ssl/certs/cert.pem
```

### Zertifikat als Secret im Cluster anwenden

`kubectl --namespace ecosystem create secret generic dogu-registry-cert --from-file=dogu-registry-cert.pem=k8s-ecosystem/cert.pem`

## Anwendung Setup

### Konfiguration values.yaml

Erstellung einer `values.yaml` mit folgendem Inhalt:

```yaml
components:
  # Use longhorn if your cluster has no storage provisioner.
  k8s-longhorn:
    version: latest
    helmRepositoryNamespace: k8s
    deployNamespace: longhorn-system
  k8s-dogu-operator:
    version: latest
    helmRepositoryNamespace: k8s
  k8s-dogu-operator-crd:
    version: latest
    helmRepositoryNamespace: k8s
  k8s-service-discovery:
    version: latest
    helmRepositoryNamespace: k8s
  k8s-snapshot-controller:
    version: latest
    helmRepositoryNamespace: k8s
  k8s-snapshot-controller-crd:
    version: latest
    helmRepositoryNamespace: k8s
  k8s-velero:
    version: latest
    helmRepositoryNamespace: k8s
  k8s-backup-operator:
    version: latest
    helmRepositoryNamespace: k8s
  k8s-backup-operator-crd:
    version: latest
    helmRepositoryNamespace: k8s
  k8s-cert-manager:
    version: latest
    helmRepositoryNamespace: k8s
  k8s-cert-manager-crd:
    version: latest
    helmRepositoryNamespace: k8s
  k8s-minio:
    version: latest
    helmRepositoryNamespace: k8s
  k8s-promtail:
    version: latest
    helmRepositoryNamespace: k8s
  k8s-loki:
    version: latest
    helmRepositoryNamespace: k8s
  k8s-ces-control:
    version: latest
    helmRepositoryNamespace: k8s
#  k8s-host-change:
#    version: latest
#    helmRepositoryNamespace: k8s

# Credentials for the docker registry used by the components.
# It is mandatory to set username and password.
docker_registry_secret:
  url: 192.168.56.10
  username: ces-admin
  password: ces-admin

# Credentials for the dogu registry used by the components.
# It is mandatory to set username and password.
dogu_registry_secret:
  url: https://192.168.56.10/nexus/repository/mirror
  username: ces-admin
  password: ces-admin
  urlschema: index

# Credentials for the helm registry used by the components.
# It is mandatory to set username and password.
helm_registry_secret:
  host: 192.168.56.10
  schema: oci
  plainHttp: "false"
  insecureTls: "true"
  username: ces-admin
  password: ces-admin

setup_json: |
  {
    "naming": {
      "fqdn": "",
      "domain": "k3ces.local",
      "certificateType": "selfsigned",
      "relayHost": "yourrelayhost.com",
      "useInternalIp": false,
      "internalIp": "",
      "completed": true
    },
    "dogus": {
      "defaultDogu": "ldap",
      "install": [
        "official/ldap",
        "official/postfix",
        "k8s/nginx-static",
        "k8s/nginx-ingress",
        "official/cas",
        "official/postgresql",
        "official/redmine"
      ],
      "completed": true
    },
    "admin": {
      "username": "ces-admin",
      "mail": "admin@admin.admin",
      "password": "ces-admin",
      "adminGroup": "cesAdmin",
      "adminMember": true,
      "sendWelcomeMail": false,
      "completed": true
    },
    "userBackend": {
      "dsType": "embedded",
      "server": "",
      "attributeID": "uid",
      "attributeGivenName": "",
      "attributeSurname": "",
      "attributeFullname": "cn",
      "attributeMail": "mail",
      "attributeGroup": "memberOf",
      "baseDN": "",
      "searchFilter": "(objectClass=person)",
      "connectionDN": "",
      "password": "",
      "host": "ldap",
      "port": "389",
      "loginID": "",
      "loginPassword": "",
      "encryption": "",
      "groupBaseDN": "",
      "groupSearchFilter": "",
      "groupAttributeName": "",
      "groupAttributeDescription": "",
      "groupAttributeMember": "",
      "completed": true
    }
  }
```

### Durchführung Setup

`helm registry login 192.168.56.10 --insecure`

`helm install k8s-ces-setup oci://192.168.56.10/k8s/k8s-ces-setup --version 0.20.1 -f values.yaml --insecure-skip-tls-verify`


