#!/usr/bin/env bash
#setup docker group based on hosts mount gid
echo "Adding hosts GID to docker system group"
# this only works if the docker group does not already exist
DOCKER_SOCKET=/var/run/docker.sock
DOCKER_GROUP=docker
BUILD_USER=go

if [ -S ${DOCKER_SOCKET} ]; then
    DOCKER_GID=$(stat -c '%g' ${DOCKER_SOCKET})

    #addgroup is distribution specific

    addgroup -S -g ${DOCKER_GID} ${DOCKER_GROUP}
    addgroup  ${BUILD_USER} ${DOCKER_GROUP}
fi
