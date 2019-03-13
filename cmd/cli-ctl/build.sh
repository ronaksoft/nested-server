#!/usr/bin/env bash
cd cli
env godep go build -o ../bin/nested-ctl
#env GOOS=linux GOARCH=amd64 go build -o ./bin/nested-ctl ./cli/
#go build -o ../nested-ctl ./
