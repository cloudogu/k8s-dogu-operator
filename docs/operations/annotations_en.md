# Annotations for the k8s-dogu-operator

This document contains the detailed description of all generated annotations of the k8s-dogu-operator.

## Services

This section contains all annotations attached to K8s services, if needed.

### k8s-dogu-operator.cloudogu.com/ces-services

The `k8s-dogu-operator.cloudogu.com/ces-services` annotation contains information about one or more CES services.
Each CES service defines an external service of the system that is accessible through the web server. The annotation is automatically
generated for each dogu that is marked as a webapp. It is also possible to customize the behavior of the services by specifying a
specifying a custom URL through which the service can be reached.

**How do I mark my dogu as a webapp?**

The requirement for your Dogu is that the `Dockerfile` provides at least one port. The dogu is marked as a webapp via
an environment variable. If the `Dockerfile` provides only one port, you have to set the environment variable
`SERVICE_TAGS=webapp` environment variable. If the `Dockerfile` contains multiple ports, it is necessary to specify the destination port of the webapp
in the environment variable. For example, we consider the exposed ports `8080,8081` and want to mark port `8081` as a
webapp. Then we need to set the environment variable `SERVICE_8081_TAGS=webapp` instead of `SERVICE_TAGS=webapp`.

**Example of a simple webapp**.

We consider the following setup:

Dockerfile
```Dockerfile
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

The Dogu operator would create a service with the following annotation "k8s-dogu-operator.cloudogu.com/ces-services":

```yaml
...
k8s-dogu-operator.cloudogu.com/ces-services: '[{"name": "my-dogu-name", "port":8080, "location":"/my-dogu-name", "pass":"/my-dogu-name"}]'
...
```

Each `k8s-dogu-operator.cloudogu.com/ces-services` entry contains an array of `ces-service` JSON objects. Each
`ces-service` object contains the following information:
* name: The name of the CES service. Used to identify the resulting ingress in the cluster.
* port: The destination port of the target service. In our case, the target service is the generated Dogu service.
* location: the URL where the CES service is accessible. Our CES service would be available in the browser as.
  `http(s)://<fqdn>/my-dogu-name`.
* pass: the URL to target in the destination server.

**Example for a webapp with additional services**.

Sometimes it is necessary to customize the `ces-service` information or even add additional services for a dogu. The
following examples explain the necessary steps. We will consider the following setup:

Dockerfile
```Dockerfile
...
ENV SERVICE_TAGS=webapp
ENV SERVICE_8080_NAME=superapp-ui
ENV SERVICE_8080_LOCATION=superapp
ENV SERVICE_ADDITIONAL_SERVICES='[{"name": "superapp-api", "port": 8080, "location": "api", "pass": "my-dogu-name/api/v2"}]'
8080,8081 EXPOSE
...
```

dogu.json
```yaml
...
"Name": "my-dogu-namespace/my-dogu-name"
...
```

The Dogu operator would create a service with the following annotation "k8s-dogu-operator.cloudogu.com/ces-services":

```yaml
...
k8s-dogu-operator.cloudogu.com/ces-services: '[{"name": "superapp-ui", "port":8080, "location":"/superapp", "pass":"/my-dogu-name"},{"name": "superapp-api", "port":8080, "location":"/api", "pass":"/my-dogu-name/api/v2"}]'
...
```

The environment variables in the form `SERVICE_<PORT>_<PROPERTY>` allow overriding the default behavior. As in our
example, the `SERVICE_8080_NAME=superapp-ui` causes the Dogu operator to create a CES service named `superapp-ui`
instead of `my-dogu-name`, which is the name of the dogu. Accepted properties are `NAME`, `LOCATION`, and `PASS`.

Besides overriding the default CES service, it is possible to add additional services. These are defined with the
environment variable `SERVICE_ADDITIONAL_SERVICES`. These can contain `ces-service` JSON objects, which are passed in the
CES-service annotation.