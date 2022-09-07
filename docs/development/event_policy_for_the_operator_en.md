# Event Policy for the k9s-dogu-operator

## What is a Kubernetes Event?

An event is a way of informing the user about processes regarding another cluster objects. A perfect example for events
is the installation of a dogu in the cluster. In general, the process information is only reported by
the `k8s-dogu-controller` in the logs. With events, it is possible to report same installation information to the
currently installed dogu resource. These events can contain any amount of information or instructions to support the
person responsible for installing the dogu in the EcoSystem.

One great strength for using events is their flexibility. It is quite easy to read the events regarding the
dogu `redmine` from the cluster and understanding its current state. It is also possible to react on specific events by
perform certain tasks for example.

More information about using events within controller can be found in
the [official documentation](https://book-v1.book.kubebuilder.io/beyond_basics/creating_events.html).