# Annotations for the k8s-dogu-operator

This document contains the detailed description of all generated annotation of the k8s-dogu-operator.

## Services

This section contains all annotation that are appended to K8s services, if necessary.

### k8s-dogu-operator.cloudogu.com/ces-services

The annotation `k8s-dogu-operator.cloudogu.com/ces-services` contains information about one or multiple ces services.
Each ces service defines an external service of the system reachable via the web server. The annotation is automatically
generated for every dogu which is marked as webapp. It is also possible to adapt the behaviour of the services, i.e., providing
custom url by which the service is available. 

**How to mark my dogu as webapp?**

The requirement for you dogu is that the `Dockerfile` exposes at least one port. The dogu is marked as webapp via
an environment variable. If the `Dockerfile` exposes only one port then you need to set the environment variable
`SERVICE_TAGS=webapp`. If the `Dockerfile` contains multiple ports it is required to mention the webapp's target port
in the environment variable. For example, we consider the exposed ports `8080,8081` and want to mark the port `8081` as
webapp. Then we need to set the environment variable `SERVICE_8081_TAGS=webapp` instead of `SERVICE_TAGS=webapp`.

**Example for a simple Webapp**

We consider the following setup:

Dockerfile
```
...
ENV SERVICE_TAGS=webapp
EXPOSE 8080
...
```

dogu.json
```yaml
...
"Name": "my-dogu-namespace/my-dogu-name"
...
```

The Dogu-Operator would create a service with the following `k8s-dogu-operator.cloudogu.com/ces-services` annotation:

```yaml
...
k8s-dogu-operator.cloudogu.com/ces-services: '[{"name":"my-dogu-name","port":8080,"location":"/my-dogu-name","pass":"/my-dogu-name"}]'
...
```

Each `k8s-dogu-operator.cloudogu.com/ces-services` contains an arrays of `ces-service` JSON objects. Every `ces-service` 
object contains the following information:
* name: The name of the ces service. Is used to identify the resulting ingress in the cluster.
* port: The target port of the target service. In our case is the target service the generated dogu service.
* location: The url where the ces-service is exposed to. Our ces-service would be available in the browser as 
  `http(s)://<fqdn>/my-dogu-name`.
* pass: The url that should targeted in the target server.

**Example for a Webapp with additional services**

Sometimes it is necessary to adapt the ces-service information or even add additional services for one dogu. The 
following examples explains the necessary steps. We consider the following setup:

Dockerfile
```
...
ENV SERVICE_TAGS=webapp
ENV SERVICE_8080_NAME=superapp-ui
ENV SERVICE_8080_LOCATION=superapp
ENV SERVICE_ADDITIONAL_SERVICES='[{"name": "superapp-api", "port": 8080, "location": "api", "pass": "my-dogu-name/api/v2"}]'
EXPOSE 8080,8081
...
```

dogu.json
```yaml
...
"Name": "my-dogu-namespace/my-dogu-name"
...
```

The Dogu-Operator would create a service with the following `k8s-dogu-operator.cloudogu.com/ces-services` annotation:

```yaml
...
k8s-dogu-operator.cloudogu.com/ces-services: '[{"name":"superapp-ui","port":8080,"location":"/superapp","pass":"/my-dogu-name"},{"name":"superapp-api","port":8080,"location":"/api","pass":"/my-dogu-name/api/v2"}]'
...
```

The environment variables in the form `SERVICE_<PORT>_<PROPERTY>` allows overwrites of the default behaviour. As in our 
example the `SERVICE_8080_NAME=superapp-ui` makes the Dogu-Operator generate a ces service with the name `superapp-ui`
instead of `my-dogu-name` which is the name of the dogu. Accepted properties are `NAME`, `LOCATION`, and `PASS`.

Besides, overwriting the default ces-service it is possible to add additional services. These are defined with the 
environment variable `SERVICE_ADDITIONAL_SERVICES`. These can contain ces-service JSON objects which are passed into
the ces-service annotation.