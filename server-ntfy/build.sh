#!/usr/bin/env bash

env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -ldflags '-s' -o ./bin/ntfy ./
docker build --pull -t registry.ronaksoftware.com/nested/server-ntfy:2.1 .
docker push registry.ronaksoftware.com/nested/server-ntfy:2.1
rm ./bin/ntfy
