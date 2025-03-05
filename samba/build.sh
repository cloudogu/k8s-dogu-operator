#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

docker build -t k3ces.local:30099/samba .
docker push k3ces.local:30099/samba
