kubectl exec --namespace ecosystem -it etcd-client -- etcdctl set "/config/ldap-mapper/backend/type" "embedded" \
&& kubectl exec --namespace ecosystem -it etcd-client -- etcdctl set "/config/ldap-mapper/backend/host" "ldap" \
&& kubectl exec --namespace ecosystem -it etcd-client -- etcdctl set "/config/ldap-mapper/backend/port" "389"