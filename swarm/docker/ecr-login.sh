#!/bin/sh

# Run this script if your Docker client is not authenticated to the AWS ECR repository
$(aws ecr get-login --no-include-email --region us-east-2)
