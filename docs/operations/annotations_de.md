# Annotationen für den k8s-dogu-operator

Dieses Dokument enthält die detaillierte Beschreibung aller erzeugten Annotationen des k8s-dogu-Operators.

## Dienste

Dieser Abschnitt enthält alle Anmerkungen, die an K8s-Dienste angehängt werden, falls erforderlich.

### k8s-dogu-operator.cloudogu.com/ces-services

Die Annotation `k8s-dogu-operator.cloudogu.com/ces-services` enthält Informationen über einen oder mehrere ces-Dienste.
Jeder ces-Dienst definiert einen externen Dienst des Systems, der über den Webserver erreichbar ist. Die Annotation wird automatisch
für jede dogu generiert, die als Webapp markiert ist. Es ist auch möglich, das Verhalten der Dienste anzupassen, d.h. eine
benutzerdefinierte Url, über die der Dienst erreichbar ist.

**Wie kennzeichne ich meine dogu als Webapp?**

Die Voraussetzung für Ihr dogu ist, dass das `Dockerfile` mindestens einen Port zur Verfügung stellt. Der dogu wird als Webapp über eine
eine Umgebungsvariable. Wenn das `Dockerfile` nur einen Port zur Verfügung stellt, müssen Sie die Umgebungsvariable
`SERVICE_TAGS=webapp` setzen. Wenn das `Dockerfile` mehrere Ports enthält, ist es erforderlich, den Zielport der Webapp
in der Umgebungsvariable anzugeben. Zum Beispiel, wir betrachten die exponierten Ports `8080,8081` und wollen den Port `8081` als
Webapp markieren. Dann müssen wir die Umgebungsvariable `SERVICE_8081_TAGS=webapp` anstelle von `SERVICE_TAGS=webapp` setzen.

**Beispiel für eine einfache Webapp**

Wir betrachten das folgende Setup:

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

Der Dogu-Operator würde einen Dienst mit der folgenden Annotation "k8s-dogu-operator.cloudogu.com/ces-services" erstellen:

```yaml
...
k8s-dogu-operator.cloudogu.com/ces-services: '[{"name":"my-dogu-name","port":8080,"location":"/my-dogu-name","pass":"/my-dogu-name"}]'
...
```

Jeder `k8s-dogu-operator.cloudogu.com/ces-services` enthält ein Array von `ces-service` JSON-Objekten. Jedes `ces-service`
Objekt enthält die folgenden Informationen:
* name: Der Name des ces-Service. Wird verwendet, um den resultierenden Ingress im Cluster zu identifizieren.
* port: Der Zielport des Zieldienstes. In unserem Fall ist der Zieldienst der generierte dogu-Dienst.
* Standort: Die URL, unter der der ces-Dienst erreichbar ist. Unser ces-Dienst wäre im Browser verfügbar als
  `http(s)://<fqdn>/mein-dogu-name`.
* pass: Die Url, die im Zielserver anvisiert werden soll.

**Beispiel für eine Webapp mit zusätzlichen Diensten**

Manchmal ist es notwendig, die ces-service Informationen anzupassen oder sogar zusätzliche Dienste für einen dogu hinzuzufügen. Die
folgenden Beispiele erklären die notwendigen Schritte. Wir betrachten das folgende Setup:

Dockerfile
```Dockerfile
...
ENV SERVICE_TAGS=webapp
ENV SERVICE_8080_NAME=superapp-ui
ENV SERVICE_8080_LOCATION=superapp
ENV SERVICE_ADDITIONAL_SERVICES='[{"name": "superapp-api", "port": 8080, "location": "api", "pass": "my-dogu-name/api/v2"}]'
8080,8081 FREILEGEN
...
```

dogu.json
```yaml
...
"Name": "my-dogu-namespace/my-dogu-name"
...
```

Der Dogu-Operator würde einen Dienst mit der folgenden Annotation "k8s-dogu-operator.cloudogu.com/ces-services" erstellen:

```yaml
...
k8s-dogu-operator.cloudogu.com/ces-services: '[{"name":"superapp-ui","port":8080,"location":"/superapp","pass":"/my-dogu-name"},{"name":"superapp-api","port":8080,"location":"/api","pass":"/my-dogu-name/api/v2"}]'
...
```

Die Umgebungsvariablen in der Form `SERVICE_<PORT>_<PROPERTY>` erlauben das Überschreiben des Standardverhaltens. Wie in unserem
Beispiel bewirkt die `SERVICE_8080_NAME=superapp-ui`, dass der Dogu-Operator einen ces-Dienst mit dem Namen `superapp-ui` erzeugt
anstelle von `my-dogu-name`, was der Name des Dogu ist. Akzeptierte Eigenschaften sind `NAME`, `LOCATION`, und `PASS`.

Neben dem Überschreiben des standardmäßigen ces-Dienstes ist es möglich, zusätzliche Dienste hinzuzufügen. Diese werden definiert mit der
Umgebungsvariable `SERVICE_ADDITIONAL_SERVICES` definiert. Diese können ces-service JSON Objekte enthalten, die in der
ces-service-Anmerkung übergeben werden.