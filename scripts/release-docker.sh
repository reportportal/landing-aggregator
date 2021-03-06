#!/bin/bash
set -e

v=$1
[[ -n "$v" ]] || read -p "Enter version (e.g. 1.0.4): " v
if [[ -z "$v" ]] ; then
	echo "Usage: $0 [<version>] (e.g. 1.0.4)"
	exit 1
fi

[[ -n "$DOCKER_USER" ]] || read -p "Enter docker user: " DOCKER_USER
if [[ -z "$DOCKER_USER" ]] ; then
	echo "Cannot process without docker user"
	exit 1
fi

[[ -n "$DOCKER_PASS" ]] || read -p "Enter docker pass: " DOCKER_PASS
if [[ -z "$DOCKER_PASS" ]] ; then
	echo "Cannot process without docker pass"
	exit 1
fi

echo "Building Docker image..."
docker build -t reportportal/landing-aggregator .
docker login -u $DOCKER_USER -p $DOCKER_PASS
docker tag reportportal/landing-aggregator reportportal/landing-aggregator:$v

echo "Pushing to DockerHUB..."
docker push reportportal/landing-aggregator:$v

echo "Deployed to DockerHUB"