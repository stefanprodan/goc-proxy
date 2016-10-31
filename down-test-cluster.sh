#!/bin/bash

image="goc-proxy"

docker rm -f $(docker ps -a -q -f "ancestor=$image")
docker rm -f $(docker ps -a -q -f "ancestor=emilevauge/whoami")
docker rmi -f $image

