# Dogu upgrades

A Dogu upgrade proceeds in the following steps:

1. the image of DoguV2 is pulled.
2. the pre-upgrade script of DoguV2 is copied to DoguV1 and executed
3. DoguV1 is shut down
4. DoguV2 is booted and waits with actual startup first
5. the post-upgrade script of DoguV2 is executed
6. DoguV2 continues its start routine

## Pre-Upgrade

### Decision-making
Unlike conventional CES, it is not so easy to copy files from an image into a running container and execute them there. Ad hoc mounting of a volume would cause a restart of the container.
This must be prevented, since the actual application must also run. With e.g. Dogus like EasyRedmine this would be
unnecessarily time-consuming.

Another idea was to extract the script via cat and insert it as a HEREDOC into the container.  
Because of the dependency on chmod and uncertainties how the HEREDOC should be passed to the Kubernetes API, 
we decided against this solution.

We also considered using a continuously running sidecar instead of the ExecPod, however this idea was rejected due 
to the waste of resources it would entail.

`kubectl cp` uses `tar` to package files and directories as archives and unpack them at the destination.
One possibility is to proceed analogously and thus not need an additional volume.

### ExecPod
The pre-upgrade script comes from the new container and is applied to the old container.  
To do this, the dogu operator starts an ExecPod of the new dogu during the upgrade and copies the script to the old dogu using `tar`.  
ExecPods use the image of the new Dogu version, but are started with Sleep Infinity.

### Running the pre-upgrade script
The pre-upgrade script is then run by the dogu operator from the `/tmp/pre-upgrade` path in the old container.

## Post-Upgrade

### Running the post-upgrade script
The Dogu operator waits until all containers of the new Dogu pod are started and then starts the post-upgrade script directly in the new Dogu pod.  
An ExecPod is not necessary, unlike in pre-upgrade, because the required script is present in the new image.

## Probes during and after the upgrade
In order to catch possible longer startup times of a dogu after an upgrade the
FailureThreshold of the startup probe is set high after an upgrade.  
After the successful upgrade this change is reset.
