#!/usr/bin/env bash

#GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build -o ./build/server-gateway
 env GOOS=linux GOARCH=amd64 go build -o ./build/server-gateway ./
 docker build --pull -t registry.ronaksoftware.com/nested/server-gateway:4.0 .
 docker push registry.ronaksoftware.com/nested/server-gateway:4.0
 rm ./build/*