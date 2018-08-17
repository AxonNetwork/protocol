#!/bin/sh

aws ecs update-service --service conscience-node --cluster conscience-node --force-new-deployment