# Asynchronous operations

For some operations it makes sense to let the `dogu-operator` wait for event asynchronously, so that it does not
does not block. For this purpose a stepper has been implemented in the `async` package which, with the help of requeueable-Errors
it is able to execute certain actions later. You can compare this stepper with a state machine.

The individual steps need a start state and deliver a final state. While the start state is fixed, the end state 
can vary depending on the event. If a step has to wait for a certain action in the cluster it can throw a
requeueable error and return its own start state. The Dogu-CR is therefore later again
reconciled and starts with the step of the state.

If a real error happened during the step, this can also be returned, so that the complete routine
aborts.

If the step is executed successfully, the start state of the next step is returned.

## Use case Adjustment Volume

![Image showing the steps to increase a dogu volume](figures/async_stepper.png)
