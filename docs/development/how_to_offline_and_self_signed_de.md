# Offline-Verwendung des k8s-CES mit selbst signierten Zertifikaten

Dieses Dokument beschreibt stichwortartig, wie das k8s-Ecosystem mit einer lokalen Docker- und Dogu-Registry aufgesetzt
werden kann. Bei der Verwendung ist zu beachten, dass die verwendete FQDN, Credentials des Ecosystems und die Dogu- bzw. Komponenten-Versionen aktuell gehalten werden.

## Aufsetzen der Dogu- und Dockerregistry
- Ecosystem aufsetzen mit Nexus-Dogu
- FQDN als insecure registry in docker config hinzufügen `/etc/docker/daemon.json`
- raw(hosted) Repository `mirror` und `k8s` anlegen

### Komponenten spiegeln:

`ces-mirror configuration.yaml`:

```yaml
version: 3

k8s:
  source:
    components:
      endpoint: https://dogu.cloudogu.com/api/v1/k8s
      username: TODO
      password: TODO
  target:
    registry:
      endpoint: 192.168.56.10
      username: ces-admin
      password: ces-admin
      insecure: true
    webserver:
      type: remote
      endpoint: https://192.168.56.10/nexus/repository/k8s
      username: ces-admin
      password: ces-admin
      insecure: true

dogu:
  dogus:
    official/postgresql:
      - 12.10-1
      - 12.13-1
    official/postfix:
      - 3.6.4-3
    official/ldap:
      - 2.6.2-3
    official/cas:
      - 6.5.8-1
    k8s/nginx-static:
      - 1.23.1-3
    k8s/nginx-ingress:
      - 1.5.1-2

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
- `go run . sync k8s`


### Dogu-Operator in Nexus ersetzen

Achtung dieser Schritt ist nicht mehr notwendig, wenn die Implementierung zur Verwendung von selbst signierten Zertifikaten
im Dogu-Operator released wurde!

Dogu-Operator yaml in Nexus ersetzen:
- Image-Tag im Makefile ändern: IMAGE_DEV=192.168.56.10/cloudogu/${ARTIFACT_ID}:${VERSION}
- `make k8s-generate`

Nexus -> browse -> k8s:
- delete k8s-dogu-operator/0.24.0
Upload Component:
- file: target/k8s-dogu-operator_0.24.0.yaml
- filename: 0.24.0
- directory: k8s/k8s-dogu-operator

## Vorbereitung k8s-Ecosystem

### K3S

-  setup.json in `k8s-ecosystem` auf completed `false` setzen
- `vagrant up`
- Zertifikat von Ecosystem `etcdctl get config/_global/certificate/servert.crt` in `k8s-ecosystem/cert.pem` speichern
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

### Konfiguration Zertifikate und Registries 

```bash
kubectl --namespace ecosystem create secret generic docker-registry-cert --from-file=docker-registry-cert.pem=cert.pem
```

- `k8s-dogu-operator-dogu-registry` und `k8s-dogu-operator-docker-registry` löschen

```bash
kubectl --namespace ecosystem create secret generic k8s-dogu-operator-dogu-registry \
--from-literal=endpoint="https://192.168.56.10/nexus/repository/mirror" \
--from-literal=username="ces-admin" \
--from-literal=password="ces-admin" \
--from-literal=urlschema="index"
```

```bash
kubectl --namespace ecosystem create secret docker-registry k8s-dogu-operator-docker-registry \
 --docker-server="192.168.56.10" \
 --docker-username="ces-admin" \
 --docker-email="myemail@test.com" \
 --docker-password="ces-admin"
```

### Setup aktualisieren

- Setup-Config bearbeiten:
```yaml
#
# The default configuration map for the ces-setup. Should always be deployed before the setup itself.
#
apiVersion: v1
kind: ConfigMap
metadata:
  name: k8s-ces-setup-config
  labels:
    app: ces
    app.kubernetes.io/name: k8s-ces-setup
data:
  k8s-ces-setup.yaml: |
    log_level: "DEBUG"
    dogu_operator_url: https://192.168.56.10/nexus/repository/k8s/k8s/k8s-dogu-operator/0.24.0
    service_discovery_url: https://192.168.56.10/nexus/repository/k8s/k8s/k8s-service-discovery/0.9.0
    etcd_server_url: https://raw.githubusercontent.com/cloudogu/k8s-etcd/develop/manifests/etcd.yaml
    etcd_client_image_repo: bitnami/etcd:3.5.2-debian-10-r0
    key_provider: pkcs1v15
```

- `make build`

Setup durchführen:
- `curl -I --request POST --url http://192.168.56.2:30080/api/v1/setup`
