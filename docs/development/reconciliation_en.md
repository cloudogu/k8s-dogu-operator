# How reconciliation works in `k8s-dogu-operator

The Dogu operator receives a Dogu resource with the target state from the Kubernetes API and passes control to it to
bring about the target state (called reconciliation). This is e.g. Dogu reinstallation, upgrade or also Dogu
uninstallation ("Operation" called).

## Requeueing to continue operations

During an operation external circumstances (like errors or long waiting times) can happen. So that the Dogu operator
does not stand still for this time, there is the possibility to terminate the Reconcile function for the time being.
This allows the operation to be executed again later.

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

If both values are empty, then the dogu resource is no longer requeued to the dogu operator by the Kubernetes API for
the current change. No more requeue takes place.

This state can occur both after a successful operation *and* on an error that cannot be handled.

### With Requeue time set `Result` and error is `nil`.

The same operation should be repeated according to its own standards. The respective operator part is responsible for
continuing an operation that has been started.

### Empty `Result` and set error

The same operation should be repeated according to Kubernetes API standards. The respective operator part is responsible
for continuing an operation that has been started.

### With Requeue time set `Result` and set error.

This behavior does not make sense. Either a `Result` or an error should be returned in the `Reconcile` function.

## What does Requeueing mean internally?

The above description shows that errors can affect the Requeueing behavior differently. To be able to make this
distinction in the individual DoguManagers (e.g. `doguInstallManager`), they use `RequeueableError` internally, which
are then evaluated `doguReconciler` as follows:

1. doguManager: no requueing by `Result` wanted, no error
    - the operation succeeded
    - no Requeue takes place
2. doguManager: no `Result`, but a `RequeueableError`
    - the operation just did not work, try again later
    - Dogu operator parts (e.g. `DoguInstallManager`) select certain errors and mark them explicitly as Requeueable
    - there is a Requeue
3. no `Result`, but a non `RequeueablerError`.
    - it will probably never work again, a manual correction is needed
    - the Dogu operator does not know how this correction should look like
    - there is no Requeue
