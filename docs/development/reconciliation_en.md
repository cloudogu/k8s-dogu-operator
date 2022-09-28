# How reconciliation works in the `k8s-dogu-operator`.

The Dogu operator receives a Dogu resource with the target state from the Kubernetes API and thus passes it the
Control to bring about the target state (called reconciliation). This can be, for example, Dogu-.
Reinstallation, Upgrade or also Dogu Uninstallation (called "operation").

## Requeueing to continue operations

During an operation, external circumstances (such as errors or long waiting times) may occur. So that the Dogu operator
does not stand still for this time, there is the possibility to stop the Reconcile function for the time being, to be executed again later.
to be executed again later.

To do this, two different hints can be given to the Kubernetes API:

- `ctrl.Result{}`, which can terminate a Requeue.
- a non-nil `error` is returned

| Result    | Error   | Requeueing            |
|-----------|---------|-----------------------|
| empty     | nil     | no                    |
| filled    | nil     | yes                   |
| empty     | error   | yes                   |
| ~~filled~ | ~error~ | not useful; see below |

### Empty `result` and error is `nil`.

If both values are empty, then the Dogu resource will not be reused by the Kubernetes API for the current
Dogu operator for the current change. No more requeue takes place.

This state can occur both after a successful operation *and* after an unhandleable error occurred.

### With Requeue time set `Result` and error is `nil`.

The same operation should be repeated according to its own standards. The respective operator part is responsible for the continuation of a started
operation.

### Empty `Result` and set error

The same operation should be repeated according to Kubernetes API standards. The respective operator part is responsible for the continuation of a started
operation.

### With a set Requeue time (in `Result`) and set error.

This behavior does not make sense. Either a `Result` or an error in the `Reconcile` function should be returned.
should be returned.

## What does Requeueing work internally?

The above description shows that errors can affect the Requeueing behavior differently. To be able to make this distinction in the individual DoguManagers (e.g. `doguInstallManager`), they use `RequeueableError` internally, which are then evaluated `doguReconciler` as follows:

1. doguManager: no requueing by `Result` wanted, no error
   - the operation succeeded
   - no Requeue takes place
2. doguManager: no `Result`, but a `RequeueableError`
   - the operation just did not work, try again later
   - Dogu operator parts (e.g. `DoguInstallManager`) mark selected errors and mark them explicitly as Requeueable
   - there will be a Requeue
3. no `Result`, but a non-`RequeueablerError`
   - it will probably never work again, a manual correction is needed
   - the Dogu operator does not know how this correction should look like
   - no Requeue takes place
