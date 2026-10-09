# Experimental Helmfile Support

As a proof of concept, the dogu operator should be able to install Helmfiles from V2 Dogus that have the `"kind": "Helmfile"` property in their `Dogu.json`.

For now, only the openDesk Helmfile will be supported.

## Prerequisites

- [Dogu.json for openDesk is pushed](#dogujson)
- [Helmfile binary image is pushed](#helmfile-binary-image)
- [HAProxy is installed](#haproxy)
- [DNS records for your domain](#dns-records)
- [ClusterIssuer for Cert-Manager is configured](#clusterissuer)
- [ConfigMap with openDesk domain is created](#opendesk-domain-configmap)

### Dogu.json

See [Dogu.json](helmfile-resources/openDesk_dogu.json) for more information.

Pushing the dogu.json:
```shell
curl --fail-with-body --basic --request PUT \
    --user "$DOGU_REGISTRY_USER" \
    --header "Content-Type: application/json" \
    --data-binary @helmfile-resources/openDesk_dogu.json \
    "https://dogu.cloudogu.com/api/v2/dogus/internal/opendesk"
```

Deleting the dogu.json:
```shell
curl --fail-with-body --basic --request DELETE \
    --user "$DOGU_REGISTRY_USER" \
    "https://dogu.cloudogu.com/api/v2/dogus/internal/opendesk/1.18.2-1"
```

### Helmfile binary image

The image can be pushed with the following make target:
```shell
make push-helmfile-image
```

### HAProxy

```shell
helm helm repo add haproxy-ingress https://haproxy-ingress.github.io/charts
helm repo update haproxy-ingress
helm upgrade --install haproxy-ingress haproxy-ingress/haproxy-ingress \
 --namespace haproxy --create-namespace -f helmfile-resources/haproxy-ingress-values.yaml
```

### DNS records

A or AAAA DNS records for `yourdomain.example.com` and `*.yourdomain.example.com` pointing to the public IP of the HAProxy LoadBalancer service must be set. Everything else is optional.

### ClusterIssuer

Since the Cloudogu Low-Ops Platform already includes cert-manager, we can just configure a ClusterIssuer.
The easiest way is usually to set up a ClusterIssuer for ACME via Let's Encrypt.
To test if it works, you can use the [staging issuer](helmfile-resources/staging-cluster-issuer.yaml).
For usable certificates, you should use the [production issuer](helmfile-resources/prod-cluster-issuer.yaml).

Make sure to set your actual domain and email address in the issuer before applying:
```shell
kubectl apply -f helmfile-resources/prod-cluster-issuer.yaml
```

### openDesk domain ConfigMap

```shell
kubectl create configmap -n ecosystem opendesk-domain --from-literal domain=yourdomain.example.com
```

## Installing openDesk as a dogu

```shell
kubectl apply -n ecosystem -f helmfile-resources/dogu-resource.yaml
```

## Debugging

Since the dogu operator image is distro-less, we have to rely on an ephemeral debug container.
To replicate volume mounts in that container and run it with elevated privileges,
we can use this [custom profile](helmfile-resources/helmfile-mount-profile.yaml).

```shell
POD_NAME=$(kubectl get pods -n ecosystem -l=k8s.cloudogu.com/component.name=k8s-dogu-operator --no-headers=true -o custom-columns=":metadata.name")
kubectl debug -n ecosystem $POD_NAME -it --image=alpine/openssl --custom=helmfile-resources/helmfile-mount-profile.yaml -- sh
```

To test the helmfile call:
```shell
cd /helmfile_data/unpack/opendesk
MASTER_PASSWORD=yourmasterpassword HELM_PLUGINS=/helmfile_binaries/helm_plugins HELM_CACHE_HOME=/helmfile_data/helm/cache HELM_DATA_HOME=/helmfile_data/helm/data HELM_CONFIG_HOME=/helmfile_data/helm/config /helmfile_binaries/helmfile apply --environment prod --namespace ecosystem --helm-binary /helmfile_binaries/helm
```