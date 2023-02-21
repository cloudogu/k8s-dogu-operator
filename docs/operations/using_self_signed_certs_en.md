## Using self-signed certificates

If the configured Docker or Dogu registry uses self-signed certificates, you must configure them using Secrets.

```bash
kubectl --namespace <cesNamespace> create secret generic docker-registry-cert --from-file=docker-registry-cert.pem=<cert_name>.pem
kubectl --namespace <cesNamespace> create secret generic dogu-registry-cert --from-file=dogu-registry-cert.pem=<cert_name>.pem
```

When the controller is restarted, the certificates are mounted to `/etc/ssl/certs/<cert_name>.pem` and are
available for used Https functions of the controller.

Additionally, the Docker Registry certificate must be distributed to all nodes in the cluster in order for Kubernetes to be able to pull images.
This may vary from distribution to distribution.

Example path Ubuntu:
`/etc/ssl/certs/<cert_name>.pem`

For k3s, it is also necessary to specify this certificate and possibly credentials in the `/etc/rancher/k3s/registries.yaml` configuration:

```yaml
configs:
  <registryURL>:
    auth:
      username: <username>
      password: <password>
    tls:
      ca_file: /etc/ssl/certs/cert.pem
```

It should be noted that credentials are not mandatory to set, because they can also be specified using the ImagePullSecrets of the
Kubernetes pod.

Afterwards, k3s must be restarted.

Main node:
```bash
systemctl restart k3s
```

Worker node:
```bash
systemctl restart k3s-agent
```
