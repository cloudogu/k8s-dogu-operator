# Event-Richtlinie für den `k8s-dogu-operator`

## Was ist ein Kubernetes-Event?

Ein Event ist eine Möglichkeit, den Benutzer über Vorgänge in Bezug auf andere Clusterobjekte zu informieren. Ein
perfektes Beispiel für ein Event ist die Installation eines Dogus im Cluster. Im Allgemeinen werden die
Prozessinformationen nur vom `k8s-dogu-controller` in den Logs gemeldet. Mit Events ist es möglich,
dieselben Installationsinformationen an die aktuell installierte Dogu-Ressource zu melden. Diese Events können eine
beliebige Menge an Informationen oder Anweisungen enthalten, um die für die Installation von Dogu im EcoSystem
verantwortliche Person zu unterstützen.

Eine große Stärke bei der Verwendung von Events ist ihre Flexibilität. Es ist recht einfach, die Events
bezüglich eines Dogus aus dem Cluster zu lesen und seinen aktuellen Zustand zu verstehen. Es ist auch möglich, auf
bestimmte Events zu reagieren, indem beliebige Aufgaben ausgeführt werden.

## Umfang der Events im `k8s-dogu-operator`

Der Dogu-Operator erstellt Events für die Dogu-CRD, während er tägliche Aufgaben wie die Installation, Löschung oder
Aktualisierung eines Dogus durchführt. Die meisten Prozessinformationen in der Dogu-Ressource helfen dabei, den
aktuellen
Prozess eines Dogus zu verstehen, ohne lange in den `k8s-dogu-operator`-Protokollen suchen zu müssen.

Ein wichtiger Faktor bei der Gestaltung neuer Prozesse und Events ist ihre Granularität. Der Betreiber sollte
wichtige Aktionen melden, damit der Benutzer den aktuellen Zustand eines Prozesses nachvollziehen kann. Das Ziel ist
jedoch nicht, kleine Vorgänge als Events zu spammen. Die folgenden Bilder zeigen eine fehlgeschlagene und eine
erfolgreiche Installation, um einen guten Überblick über die Granularität des `k8s-dogu-operator` zu erhalten.

**Fehler bei der `redmine`-Dogu-Installation**

![Bild, das die Ereignisse beim Auftreten eines Fehlers bei der Installation des `postgresql` Dogu zeigt.](figures/events_with_errors.png)

**Erfolgreiche `postgresql`-Dogu-Installation**

![Bild mit Ereignissen für die erfolgreiche Installation des `postgresql` Dogu.](figures/events_without_errors.png)

## Verwendung von Ereignissen im `k8s-dogu-operator`

Events können mit einem [EventRecoder](https://pkg.go.dev/k8s.io/client-go/tools/record#EventRecorder) an 
Dogu-Ressourcen geschrieben werden. Der Manager implementiert das Interface 
[Cluster](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/cluster#Cluster). Die darin enthaltene Methode
`GetEventRecorderFor(name string)` liefert eine Referenz zu einem EventRecorder-Objekt.
Die Verwendung kann zum Beispiel in dem Installation-Manager des `k8s-dogu-operator` betrachtet werden.
