#!/usr/bin/env bash

## Define Variables
GATEWAY_VER=4.0
NTFY_VER=3.0
MTA_VER=2.0

cd ./cmd

## Build Server Gateway
cd ./server-gateway/
env GOOS=linux GOARCH=amd64 go build -o ./_build/server-gateway ./
docker build --pull -t registry.ronaksoftware.com/nested/server-gateway:${GATEWAY_VER} .
#docker push registry.ronaksoftware.com/nested/server-gateway:${GATEWAY_VER}
cd ..

## Build Server NTFY
cd ./server-ntfy/
env GOOS=linux GOARCH=amd64 go build -o ./_build/server-ntfy ./
docker build --pull -t registry.ronaksoftware.com/nested/server-ntfy:${NTFY_VER} .
#docker push registry.ronaksoftware.com/nested/server-ntfy:${NTFY_VER}
cd ..

## Build Server MTA
cd ./server-mta/
env GOOS=linux GOARCH=amd64 go build -o ./_build/mail-store-cli ./mail-store-cli/
env GOOS=linux GOARCH=amd64 go build -o ./_build/mail-map ./mail-map/
docker build --pull -t registry.ronaksoftware.com/nested/server-mta:${MTA_VER} .
#docker push registry.ronaksoftware.com/nested/server-mta:${MTA_VER}
cd ..