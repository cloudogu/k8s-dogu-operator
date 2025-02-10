#!/bin/bash

docker build -t k3ces.local:30099/dogu-rsync-sidecar .
docker push k3ces.local:30099/dogu-rsync-sidecar
