#!/usr/bin/env bash

## Define Variables
GATEWAY_VER=4.0
NTFY_VER=3.0
MTA_VER=2.0

## Build Server Gateway
env GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build -o ./_build/server-gateway ./
#docker build --pull -t registry.ronaksoftware.com/nested/server-gateway:${GATEWAY_VER} .
#docker push registry.ronaksoftware.com/nested/server-gateway:${GATEWAY_VER}
