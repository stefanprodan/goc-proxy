#!/bin/bash
set -e

hostIP="$(hostname -I|awk '{print $1}')"

# consul 
docker run -d \
-p $hostIP:8500:8500 \
-p $hostIP:8400:8400 \
-p $hostIP:8300:8300 \
-v /home/$(whoami)/consul:/consul/data \
-e SERVICE_IGNORE=true \
--name consul \
--restart unless-stopped \
consul \
agent -server -ui -bootstrap-expect=1 -advertise=$hostIP -client=0.0.0.0

# registrator
docker run -d \
--name=registrator \
--net=host \
--volume=/var/run/docker.sock:/tmp/docker.sock \
--restart unless-stopped \
gliderlabs/registrator -ip="${hostIP}" consul://${hostIP}:8500 
