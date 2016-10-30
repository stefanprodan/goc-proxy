#!/bin/bash
set -e

image="goc-proxy"

# build image
if [ ! "$(docker images -q  $image)" ];then
    docker build -t $image .
fi

hostIP="$(hostname -I|awk '{print $1}')"

# start proxy
docker run -d -p 8000:8000 \
--name "$image" \
--restart unless-stopped \
--ulimit nofile=65536:65536 \
-e CONSUL_HTTP_ADDR="${hostIP}:8500" \
-e SERVICE_CHECK_HTTP="/_/status" \
-e SERVICE_CHECK_INTERVAL="15s" \
$image goc-proxy \
-Environment=TEST \
-Port=8000 \
-Loglevel=info \
-MaxIdleConnsPerHost=10000 

# start alpha services with one node
docker run -d -P \
-h alpha-node1 \
--name alpha-node1 \
-e SERVICE_NAME="alpha" \
-e SERVICE_CHECK_HTTP="/api" \
-e SERVICE_CHECK_INTERVAL="15s" \
emilevauge/whoami

# start beta services with 2 nodes
for ((i=1; i<3; i++)); do

docker run -d -P \
-h beta-node$i \
--name beta-node$i \
-e SERVICE_NAME="beta" \
-e SERVICE_CHECK_HTTP="/api" \
-e SERVICE_CHECK_INTERVAL="15s" \
emilevauge/whoami

done

# start gamma services with 3 nodes
for ((i=1; i<4; i++)); do

docker run -d -P \
-h gamma-node$i \
--name gamma-node$i \
-e SERVICE_NAME="gamma" \
-e SERVICE_CHECK_HTTP="/api" \
-e SERVICE_CHECK_INTERVAL="15s" \
emilevauge/whoami

done

