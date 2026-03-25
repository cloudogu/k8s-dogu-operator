# Exposed Ports

Exposed ports of Dogus must be publicly available so that users can use services such as the Git protocol on port 2222
for the SCM-Manager.
This process is different from the old Cloudogu EcoSystem where the port was shared on the host and routed to the
appropriate service.

# Expose HTTP ports

In Kubernetes, the `k8s-dogu-operator` creates **one** service `ces-loadblancer` of type LoadBalancer for this purpose.
In most cloud infrastructures, a publicly accessible IP is assigned to this service, which serves as the entry point to the
Cloudogu EcoSystem.
Typically, port `80` and `443` are always included in this service, as they are defined by the reverse proxy to ensure
that other Dogus are reachable using ingress resources.

# Expose TCP/UDP ports

Other ports, such as `2222` for the SCM-Manager, are not necessarily based on the HTTP protocol and use
pure TCP or UDP. These ports are also entered into the `ces-loadblancer`.
In Traefik, `IngressRouteTCP` or `IngressRouteUPD` resources are dynamically created for these ports by the
`k8s-ces-gateway`. Traefik middlewares handle the routing.
