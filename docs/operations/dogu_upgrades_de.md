# Dogu-Upgrades

Ein Dogu-Upgrade stellt auf den ersten Blick nicht mehr dar, als eine neue Dogu-Version in das Cloudogu EcoSystem einspielen. Ein Dogu-Upgrade ist eine von mehreren Operationen, die `k8s-dogu-operator` unterstützt. Grundsätzlich ist es nur möglich, Dogus mit einer höheren Version zu aktualisieren. Sonderfälle diskutiert der Abschnitt "Upgrade-Sonderfälle"

Ein solches Upgrade lässt sich leicht erzeugen.

**Beispiel:**

Ein Dogu wurde bereits in einer älteren Version mit dieser Dogu-Resource mittels `kubectl apply` installiert:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: my-dogu
  labels:
    dogu: my-dogu
    app: ces
spec:
  name: official/my-dogu
  version: 1.2.3-4
```

Ein Upgrade des Dogus auf Version `1.2.3-5` ist denkbar einfach. Eine vergleichbare Resource mit neuerer Version herstellen und wieder mit `kubectl apply ...` auf den Cluster anwenden:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Dogu
metadata:
  name: my-dogu
  labels:
    dogu: my-dogu
    app: ces
spec:
  name: official/my-dogu
  version: 1.2.3-5
```

## Pre-Upgrade-Skripte

To Do PUS

## Upgrade-Sonderfälle

### Downgrades

To Do DG

### Wechsel eines Dogu-Namespaces

To Do DNW