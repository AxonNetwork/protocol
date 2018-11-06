#!/bin/bash

__dirname="$(CDPATH= cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

docker build -t conscience-node --file "${__dirname}/Dockerfile.node" "${__dirname}/../.."

