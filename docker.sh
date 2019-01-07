#! /usr/bin/env bash

function error {
	echo ERROR: "$1"
	exit 1
}

command -v docker >/dev/null 2>&1 || error "docker not installed"

DOCKER_IMAGE="syscll/elasticipd:latest"

docker build -t $DOCKER_IMAGE . || error "failed to build Docker image"
docker login -u $DOCKER_USER -p $DOCKER_PASSWORD
docker push $DOCKER_IMAGE || error "failed to distribute Docker image"
