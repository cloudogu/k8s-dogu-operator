#!/bin/bash
namespace=$1
doguName=$2
cmName="${doguName}-config"
kubectl create cm "${cmName}" --from-file=config.yaml="${doguName}_config.yaml" -n "${namespace}"
kubectl label configmap "${cmName}" app=ces -n "${namespace}"
kubectl label configmap "${cmName}" dogu.name="${doguName}" -n "${namespace}"
kubectl label configmap "${cmName}" k8s.cloudogu.com/type=dogu-config -n "${namespace}"