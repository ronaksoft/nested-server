#!/usr/bin/env bash

# Install docker pre-requisite packages
sudo apt-get install -y apt-transport-https ca-certificates \
    curl software-properties-common sysfsutils rsync libltdl7

# Install docker
dpkg -i ./debs/docker-ce_17.06.1-ce-0-ubuntu_amd64.deb

# Install docker-compose and set the permissions
cp ./debs/docker-compose /usr/local/bin/
chmod 755 /usr/local/bin/docker-compose