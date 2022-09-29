# Wie Reconciliation im `k8s-dogu-operator` funktioniert

Der Dogu-Operator erhält von der Kubernetes-API eine Dogu-Resource mit dem Zielzustand und übergibt ihm damit die
Steuerung zur Herbeiführung des Zielzustandes ("Reconciliation" genannt). Dabei handelt es sich z. B. um
Dogu-Neuinstallation, Upgrade oder auch Dogu-Deinstallation ("Operation" genannt).

## Requeueing, um Operationen fortzuführen

Während einer Operation können äußere Umstände (wie Fehler oder lange Wartezeiten) geschehen. Damit der Dogu-Operator
für diese Zeit nicht still steht, besteht die Möglichkeit, die Reconcile-Funktion vorerst zu beenden. Die Operation kann
damit später erneut ausgeführt zu werden.

Dazu können der Kubernetes-API zwei unterschiedliche Hinweise gegeben werden:

- `ctrl.Result{}`, das ein Requeue terminieren kann
- ein nicht-nil `error` wird zurückgegeben

| Result      | Fehler    | Requeueing            |
|-------------|-----------|-----------------------|
| leer        | nil       | nein                  |
| gefüllt     | nil       | ja                    |
| leer        | error     | ja                    |
| ~~gefüllt~~ | ~~error~~ | nicht sinnvoll; s. u. |

### Leeres `Result` und Fehler ist `nil`

Wenn beide Werte leer sind, dann wird die Dogu-Resource nicht mehr erneut von der Kubernetes-API für die aktuelle
Änderung an den Dogu-Operator übergeben. Es findet kein Requeue mehr statt.

Dieser Zustand kann sowohl nach einer erfolgreichen Operation *als auch* bei einem nicht behandelbaren Fehler eintreten.

### Mit Requeue-Zeit gesetztes `Result` und Fehler ist `nil`

Die gleiche Operation sollte nach eigenen Maßstäben wiederholt werden. Der jeweilige Operatorteil ist für die
Fortführung einer begonnenen
Operation verantwortlich.

### Leeres `Result` und gesetzter Fehler

Die gleiche Operation sollte nach Maßstäben der Kubernetes-API wiederholt werden. Der jeweilige Operatorteil ist für die
Fortführung einer begonnenen
Operation verantwortlich.

### Mit Requeue-Zeit gesetztes `Result` und gesetzter Fehler

Dieses Verhalten ergibt keinen Sinn. Es sollte entweder ein `Result` oder ein Fehler in der `Reconcile`-Funktion
zurückgegeben werden.

## Was bedeutet Requeueing intern?

Obige Beschreibung zeigt, dass sich Fehler unterschiedlich auf das Requeuing-Verhalten ausüben können. Um diese
Unterscheidung in den einzelnen DoguManagern (z. B. `doguInstallManager`) treffen zu können, verwenden diese
intern `RequeueableError`, die dann `doguReconciler` wiefolgt ausgewertet werden:

1. DoguManager: kein Requeueing durch `Result` gewünscht, kein Fehler
    - die Operation ist gelungen
    - es findet kein Requeue statt
2. DoguManager: kein `Result`, aber ein `RequeueableError`
    - die Operation hat gerade nicht funktioniert, es soll später noch einmal versucht werden
    - Dogu-Operator-Teile (z. B. `DoguInstallManager`) wählen bestimmte Fehler aus und markieren diese explizit als
      Requeueable
    - es findet ein Requeue statt
3. kein `Result`, aber ein Nicht-`RequeueablerError`
    - es wird wohl nie wieder funktionieren, es ist eine manuelle Korrektur nötig
    - der Dogu-Operator weiß nicht, wie diese Korrektur aussehen soll
    - es findet kein Requeue statt
