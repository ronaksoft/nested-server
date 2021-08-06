#!/usr/bin/env bash

## Define Variables
NESTED_VER=1.0

buildDir=./cmd/_build
mkdir -p $buildDir

## Build CLI_API
execName=cli-api
env GOOS=linux GOARCH=amd64 go build -o $buildDir/$execName ./cmd/$execName
## Build CLI_MAIL_MAP
execName=cli-mail-map
env GOOS=linux GOARCH=amd64 go build -o $buildDir/$execName ./cmd/$execName


docker build --pull -t registry.ronaksoft.com/nested/server:${NESTED_VER} .

