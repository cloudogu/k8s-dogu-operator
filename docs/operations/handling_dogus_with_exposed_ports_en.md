# Exposed Ports

Exposed ports of Dogus must be publicly available so that users can use services such as the Git protocol on port 2222
for the SCM Manager.
This process is different from the old Cloudogu EcoSystem where the port was shared on the host and routed to the
appropriate service.

# Expose HTTP ports

In Kubernetes, the `k8s-dogu-operator` creates **one** service `ces-loadblancer` of type LoadBalancer for this purpose.
In most cloud infrastructures, a publicly accessible IP is assigned to this service, which serves as the entry point to the
Cloudogu EcoSystem.
Typically, port `80` and `443` are always included in this service, as they are defined by `nginx-ingress` to ensure
that other Dogus are reachable using ingress resources.

# Expose TCP/UDP ports

Other ports, such as `2222` for the SCM manager, are not necessarily based on the HTTP protocol and use
pure TCP or UDP. These ports are also entered into the `ces-loadblancer`. Additionally, these
are written to `tcp-services` and `udp-services` configmaps for the `nginx-ingress` to route the traffic
(see [nginx-guide](https://kubernetes.github.io/ingress-nginx/user-guide/exposing-tcp-udp-services/)).
The configmaps each have a finalizer `cloudogu.com/nginx-tcp-services` so that they are not be deleted by
accident. The format for an exposed service in such a configmap is the following:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: tcp-services
  namespace: ingress-nginx
data:
  2222: "ecosystem/scm:2222"
```

The key maps the port of the host. The value consists of the namespace, service name, and the container port
to which the traffic should be forwarded.