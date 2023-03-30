# Event Policy for the `k8s-dogu-operator`

## What is a Kubernetes Event?

An event is a way of informing the user about processes regarding another cluster objects. A perfect example for events
is the installation of a dogu in the cluster. In general, the process information is only reported by
the `k8s-dogu-controller` in the logs. With events, it is possible to report same installation information to the
currently installed dogu resource. These events can contain any amount of information or instructions to support the
person responsible for installing the dogu in the EcoSystem.

One great strength for using events is their flexibility. It is quite easy to read the events regarding the
dogu `redmine` from the cluster and understanding its current state. It is also possible to react on specific events by
perform any tasks.

## Scope of Events in the `k8s-dogu-operator`

The dogu operator creates events for the dogu CRD while performing daily tasks such as installing, deleting or upgrading
a dogu. Having most of the process information on the dogu resource helps to understand the current process of a
dogu without having to search in the `k8s-dogu-operator` logs for an extended time.

An important factor when designing new processes and events is their granularity. The operator should report significant
actions so that the user can understand the current state of a process. However, the aim is not to spam small operation
as an event. The following images show an unsuccessful installation and a successful one to get a good
overview about the granularity used in the `k8s-dogu-operator`.

**Error on `redmine` dogu installation**

![Image depicting events when error on `postgresql` dogu installation occured.](figures/events_with_errors.png)

**Successful `postgresql` dogu installation**

![Image depicting events for successful `postgresql` dogu installation.](figures/events_without_errors.png)

## Using events in the `k8s-dogu-operator`
<!-- TODO Please check this link because actual kubebuilder has ssl issues -->
<!-- markdown-link-check-disable -->
The [kubebuilder documentation](https://book-v1.book.kubebuilder.io/beyond_basics/creating_events.html) explains
perfectly how to use events inside a kubernetes controller.