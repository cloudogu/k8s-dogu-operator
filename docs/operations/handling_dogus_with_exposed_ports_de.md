# Exposed Ports

Exposed Ports von Dogus müssen öffentlich zugänglich sein, damit User Services wie zum Beispiel das Git-Protokoll auf
Port 2222 für den SCM-Manager verwenden können.
Dieser Vorgang unterscheidet sich zum alten Cloudogu EcoSystem, bei dem der Port auf dem Host freigegeben und zum
entsprechenden Service geroutet wurde.

# Expose HTTP-Ports

In Kubernetes erstellt der `k8s-dogu-operator` dafür **einen** Service `ces-loadblancer` vom Typ LoadBalancer. Diesem
Service wird in den meisten Cloud-Infrastrukturen eine öffentlich zugängliche IP zugewiesen, die als Zugang zum Cloudogu
EcoSystem dient.
Typischerweise sind die Ports `80` und `443` immer in diesem Service enthalten, da diese vom `reverse proxy` definiert
werden und dafür sorgen, dass andere Dogus mithilfe von Ingress-Ressourcen erreichbar sind.

# Expose TCP/UDP-Ports

Andere Ports, wie zum Beispiel `2222` für den SCM-Manager, basieren nicht zwingend auf dem HTTP-Protokoll und verwenden
reines TCP oder UDP.
Diese Ports werden ebenfalls in den `ces-loadblancer` eingetragen.

In Traefik werden für diese Ports jeweils `IngressRouteTCP` oder `IngressRouteUPD` Ressourcen dynamisch durch das 
`k8s-ces-gateway` erstellt. Eine Traefik Middleware übernimmt das Routing.