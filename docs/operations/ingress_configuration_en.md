# Ingress Configuration

Ingress rules for dogus get generated by [k8s-service-discovery](https://github.com/cloudogu/k8s-service-discovery) and should not be edited manually.  
However, a lot of configuration can be done through annotations on ingress rules.

## Ingress annotations
Since ingress rules for dogus should not be edited manually, [ingress annotations](https://docs.nginx.com/nginx-ingress-controller/configuration/ingress-resources/advanced-configuration-with-annotations/) can be added to the ingress rules through the dogu resource.  
Simply add them in the field `additionalIngressAnnotations` of the dogu resources `spec` field.

Example:
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
    nginx.org/client-max-body-size: "0"
```