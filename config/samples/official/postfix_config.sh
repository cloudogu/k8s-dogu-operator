kubectl exec --namespace ecosystem -it etcd-client -- etcdctl set "config/postfix/relayhost" "mail.mydomain.com"