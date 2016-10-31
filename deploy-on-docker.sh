#!/bin/bash
set -e

read -p "Enter domain: " domain

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
-LogLevel=info \
-Port=8000 \
-MaxIdleConnsPerHost=10000 \
-Domain=$domain

sleep 1
info="$(curl -fsSL "http://${hostIP}:8000/_/status")"
echo "goc-proxy status"
echo $info

