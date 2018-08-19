#!/bin/sh

aws ecs update-service --service swarm-node-conscience-node --cluster swarm-node --force-new-deployment
