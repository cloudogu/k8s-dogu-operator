kubectl exec -it etcd-client -- etcdctl set "config/cas/ldap/ds_type" "embedded" \
&& kubectl exec -it etcd-client -- etcdctl set "/config/cas/ldap/attribute_id" "uid" \
&& kubectl exec -it etcd-client -- etcdctl set "/config/cas/ldap/attribute_group" "memberof" \
&& kubectl exec -it etcd-client -- etcdctl set "/config/cas/ldap/attribute_mail" "mail" \
&& kubectl exec -it etcd-client -- etcdctl set "/config/cas/ldap/search_filter" "(objectClass=person)" \
&& kubectl exec -it etcd-client -- etcdctl set "/config/cas/ldap/attribute_fullname" "cn" \
&& kubectl exec -it etcd-client -- etcdctl set "/config/cas/ldap/encryption" "none" \
&& kubectl exec -it etcd-client -- etcdctl set "/config/cas/ldap/host" "ldap" \
&& kubectl exec -it etcd-client -- etcdctl set "/config/cas/ldap/port" "389" \
&& kubectl exec -it etcd-client -- etcdctl set "/config/cas/logging/root" "DEBUG"