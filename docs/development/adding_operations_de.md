# Operationen zum Operator hinzufügen

Wenn die Custom-Resource um Felder erweitert wird, welche im Operator zu einer neuen Operation führen sollen,
dann muss der Operator an folgenden Stellen angepasst werden:
- `evaluateRequiredOperation()` bzw. `appendRequiredPostInstallOperations()` muss angepasst werden, sodass die Operation erkannt wird.
  Zu beachten ist hier, dass der Operator mehrere Operationen erkennen kann. Somit muss die Operation an der passenden Stelle eingefügt werden.
- In `executeRequiredOperation()` werden die Operationen dann ausgewertet und ausgeführt. 
  Es wird immer nur die erste erkannte Operation ausgeführt. Wenn es mehr als eine Operation gibt, wird ein Requeue veranlasst, um die anderen Operationen auszuführen.
