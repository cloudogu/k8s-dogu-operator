#!/bin/bash
etcdClient="$(kubectl get pods -l=app.kubernetes.io/name=etcd-client -o name 2>&1 | head -n 1)"

kubectl exec --namespace ecosystem "$etcdClient" -- etcdctl set "config/cas/ldap/ds_type" "embedded" \
&& kubectl exec --namespace ecosystem "$etcdClient" -- etcdctl set "/config/cas/ldap/attribute_id" "uid" \
&& kubectl exec --namespace ecosystem "$etcdClient" -- etcdctl set "/config/cas/ldap/attribute_group" "memberof" \
&& kubectl exec --namespace ecosystem "$etcdClient" -- etcdctl set "/config/cas/ldap/attribute_mail" "mail" \
&& kubectl exec --namespace ecosystem "$etcdClient" -- etcdctl set "/config/cas/ldap/search_filter" "(objectClass=person)" \
&& kubectl exec --namespace ecosystem "$etcdClient" -- etcdctl set "/config/cas/ldap/attribute_fullname" "cn" \
&& kubectl exec --namespace ecosystem "$etcdClient" -- etcdctl set "/config/cas/ldap/encryption" "none" \
&& kubectl exec --namespace ecosystem "$etcdClient" -- etcdctl set "/config/cas/ldap/host" "ldap" \
&& kubectl exec --namespace ecosystem "$etcdClient" -- etcdctl set "/config/cas/ldap/port" "389" \
&& kubectl exec --namespace ecosystem  "$etcdClient" -- etcdctl set "/config/cas/logging/root" "DEBUG"