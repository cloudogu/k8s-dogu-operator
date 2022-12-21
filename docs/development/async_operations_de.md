# Asynchrone Operationen

Für manche Operationen ist es sinnvoll den `dogu-operator` asynchron auf Ereignis warten zu lassen, damit dieser
nicht blockiert. Dazu wurde im `async` Package ein Stepper implementiert der Mithilfe von requeueable-Errors
in der Lage ist bestimmte Aktionen später neu auszuführen. Vergleichen kann man diesen Stepper mit einer Zustandsmaschine.

Die einzelnen Steps benötigen einen Startzustand und liefern einen Endzustand. Während der Startzustand fix ist, kann 
der Endzustand je nach Ereignis variieren. Muss ein Step auf eine bestimmte Aktion im Cluster warten kann er so lange ein
requeueable-Error werfen und seinen eigenen Startzustand zurückgeben. Die Dogu-CR wird demnach später wieder 
reconciled und startet mit dem Step des Zustands.

Ist während des Steps ein echter Fehler passiert, kann dieser ebenfalls zurückgegeben werden, damit die komplette Routine
abbricht.

Bei der erfolgreichen Durchführung des Steps wird der Startzustand des nächsten Steps zurückgegeben.

## Anwendungsfall Anpassung Volume

![Bild, das die Steps zur Vergrößerung eines Dogu-Volumes zeigt.](figures/async_stepper.png)

