## Verwendung von selbst signierten Zertifikaten

Falls die konfigurierte Docker- oder Dogu-Registry selbst signierte Zertifikate verwenden, muss man diese Mithilfe von Secrets
konfigurieren.

```bash
kubectl --namespace <cesNamespace> create secret generic docker-registry-cert --from-file=docker-registry-cert.pem=<cert_name>.pem
kubectl --namespace <cesNamespace> create secret generic dogu-registry-cert --from-file=dogu-registry-cert.pem=<cert_name>.pem
```

Bei einem Neustart des Controllers werden die Zertifikate nach `/etc/ssl/certs/<cert_name>.pem` gemountet und sind
für verwendeten Http-Funktionen des Controllers verfügbar.

Zusätzlich muss das Zertifikat der Docker-Registry auf allen Nodes des Clusters verteilt werden, damit Kubernetes Images pullen kann.
Dies kann von Distribution zu Distribution unterschiedlich sein.

Beispiel-Pfad Ubuntu:
`/etc/ssl/certs/<cert_name>.pem`

Bei k3s ist es außerdem erforderlich dieses Zertifikat und möglicherweise Credentials in der Konfiguration `/etc/rancher/k3s/registries.yaml` anzugeben:

```yaml
configs:
  <RegistryURL>:
    auth:
      username: <username>
      password: <password>
    tls:
      ca_file: /etc/ssl/certs/cert.pem
```

Es ist zu beachten, dass Credentials nicht zwingend gesetzt werden müssen, weil diese auch über die ImagePullSecrets des
Kubernetes-Pods angegeben werden können.

Anschließend muss k3s neu gestartet werden.

Main-Node:
```bash
systemctl restart k3s
```

Worker-Node:
```bash
systemctl restart k3s-agent
```
