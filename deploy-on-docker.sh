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
-ServiceName=$image \
-ClusterName=cl1 \
-Environment=TEST \
-Loglevel=info \
-Port=8000 \
-MaxIdleConnsPerHost=10000 

sleep 1
info="$(curl -fsSL "http://${hostIP}:8000/_/status")"
echo "goc-proxy status"
echo $info

