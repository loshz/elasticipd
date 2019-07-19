#! /usr/bin/env bash

set -o errexit

DOCKER_IMAGE="syscll/elasticipd:latest"

docker build --tag $DOCKER_IMAGE .
docker push $DOCKER_IMAGE
