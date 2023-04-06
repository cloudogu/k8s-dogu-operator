# Ingress-Konfiguration

Ingress-Regeln für Dogus werden von [k8s-service-discovery](https://github.com/cloudogu/k8s-service-discovery) generiert und sollten nicht manuell bearbeitet werden.  
Allerdings kann ein Großteil der Konfiguration durch Anmerkungen zu den Ingress-Regeln vorgenommen werden.

## Ingress-Annotationen
Da die Ingress-Regeln für Dogus nicht manuell bearbeitet werden sollten, können [NGINX-Ingress-Annotationen](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/) über die Dogu-Ressource zu den Ingress-Regeln hinzugefügt werden.  
Sie werden einfach in das Feld `additionalIngressAnnotations` im `spec`-Feld der Dogu-Ressource angehängt.

Beispiel:
```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: nexus
  labels:
    app: ces
spec:
  name: official/nexus
  version: 3.40.1-2
  additionalIngressAnnotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "0"
```