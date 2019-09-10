#!/bin/sh
docker build -t axon-geth --file ./Dockerfile.geth .
docker run -t -p 8545:8545 -p 8546:8546 -p 30303:30303 axon-geth
