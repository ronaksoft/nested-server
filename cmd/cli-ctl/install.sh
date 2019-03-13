#!/usr/bin/env bash
#####################
# The user running this script must be SU-DOER
###############
INSTALL_PATH=/ronak/nested

# Copy binary files to PATH folder
cp ./bin/* /usr/local/bin/

# Create nested folder
mkdir -p ${INSTALL_PATH}

# Copy templates folder
cp -r ./templates/ ${INSTALL_PATH}

# Copy config dir with samples
cp -r ./config ${INSTALL_PATH}

# Docker login to registry
docker login registry.ronaksoftware.com -u docker -p sLxDKdMuNpit_dL3YTPg

# Prepare system for MongoDB
# Reboot required
echo kernel/mm/transparent_hugepage/enabled=never >> /etc/sysfs.conf
echo kernel/mm/transparent_hugepage/defrag=never >> /etc/sysfs.conf
