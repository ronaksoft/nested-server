#!/usr/bin/env bash

## Define Variables
GATEWAY_VER=4.0
NTFY_VER=3.0
MTA_VER=2.0


## Build Server Gateway
cd ./cmd/server-gateway/ || exit
env GOOS=linux GOARCH=amd64 go build -o ./_build/server-gateway ./
docker build --pull -t registry.ronaksoft.com/nested/server-gateway:${GATEWAY_VER} .
cd ../..

## Build Server NTFY
cd ./cmd/server-ntfy/ || exit
env GOOS=linux GOARCH=amd64 go build -o ./_build/server-ntfy ./
docker build --pull -t registry.ronaksoft.com/nested/server-ntfy:${NTFY_VER} .
cd ../..

## Build Server MTA
cd ./cmd/server-mta/ || exit
env GOOS=linux GOARCH=amd64 go build -o ./_build/mail-store-cli ./mail-store-cli/
env GOOS=linux GOARCH=amd64 go build -o ./_build/mail-map ./mail-map/
docker build --pull -t registry.ronaksoft.com/nested/server-mta:${MTA_VER} .
cd ../..