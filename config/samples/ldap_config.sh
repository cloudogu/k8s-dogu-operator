kubectl exec -it etcd-client -- etcdctl set "config/ldap/admin_mail" "mail@test.de" \
&& kubectl exec -it etcd-client -- etcdctl set "config/ldap/admin_member" "true"