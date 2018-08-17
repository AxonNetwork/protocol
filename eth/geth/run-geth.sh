#!/bin/sh
docker build -t conscience-geth --file ./Dockerfile.geth .
docker run -t -p 8545:8545 -p 30303:30303 conscience-geth