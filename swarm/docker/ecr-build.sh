#!/bin/bash

__dirname="$(CDPATH= cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# build the protocol binary
"${__dirname}/../../make.sh" --docker && \

# download dumb-init
wget -O "${__dirname}/../../build/docker/dumb-init" https://github.com/Yelp/dumb-init/releases/download/v1.2.2/dumb-init_1.2.2_amd64 && \

# build the Docker image
docker build -t axon-node --file "${__dirname}/Dockerfile.node" "${__dirname}/../../build/docker"

