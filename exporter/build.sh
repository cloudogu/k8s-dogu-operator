#!/bin/bash

docker build -t k3ces.local:30099/exporter .
docker push k3ces.local:30099/exporter
