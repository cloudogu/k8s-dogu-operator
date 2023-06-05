# Exposed Ports

Exposed Ports von Dogus müssen öffentlich zugänglich sein, damit User Services wie zum Beispiel das Git-Protokoll auf
Port 2222 für den SCM-Manager verwenden können.
Dieser Vorgang unterscheidet sich zum alten Cloudogu EcoSystem, bei dem der Port auf dem Host freigegeben und zum
entsprechenden Service geroutet wurde.

# Expose HTTP-Ports

In Kubernetes erstellt der `k8s-dogu-operator` dafür **einen** Service `ces-loadblancer` vom Typ LoadBalancer. Diesem
Service wird in den meisten Cloud-Infrastrukturen eine öffentlich zugängliche IP zugewiesen, die als Zugang zum Cloudogu
EcoSystem dient.
Typischerweise sind die Ports `80` und `443` immer in diesem Service enthalten, da diese von `nginx-ingress` definiert
werden und dafür sorgen, dass andere Dogus mithilfe von Ingress-Ressourcen erreichbar sind.

# Expose TCP/UDP-Ports

Andere Ports, wie zum Beispiel `2222` für den SCM-Manager, basieren nicht zwingend auf dem HTTP-Protokoll und verwenden
reines TCP oder UDP.
Diese Ports werden ebenfalls in den `ces-loadblancer` eingetragen.
Zusätzlich werden diese in Configmaps `tcp-services` und `udp-services` für den `nginx-ingress` geschrieben, damit
dieser den Traffic routen kann (
siehe [nginx-guide](https://kubernetes.github.io/ingress-nginx/user-guide/exposing-tcp-udp-services/)).
Die Configmaps besitzen jeweils einen Finalizer `cloudogu.com/nginx-tcp-services`, damit diese nicht aus Versehen
gelöscht werden.
Das Format für einen exposed Service in einer solchen Configmap ist Folgendes:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: tcp-services
  namespace: ingress-nginx
data:
  2222: "ecosystem/scm:2222"
```

Der Key bildet den Ports des Hosts ab.
Der Wert besteht aus dem Namespace, Servicenamen und dem Containerport zu dem der Traffic weitergeleitet werden soll.