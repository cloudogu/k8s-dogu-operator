#!/bin/bash
etcdClient="$(kubectl get pods -l=app.kubernetes.io/name=etcd-client -o name 2>&1 | head -n 1)"

kubectl exec --namespace ecosystem -it "$etcdClient" -- etcdctl set "/config/ldap-mapper/backend/type" "embedded" \
&& kubectl exec --namespace ecosystem -it "$etcdClient" -- etcdctl set "/config/ldap-mapper/backend/host" "ldap" \
&& kubectl exec --namespace ecosystem -it "$etcdClient" -- etcdctl set "/config/ldap-mapper/backend/port" "389"