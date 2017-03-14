# Landing Page Info Aggregator

[![Build Status](https://travis-ci.org/reportportal/landing-aggregator.svg?branch=master)](https://travis-ci.org/reportportal/landing-aggregator)
[![Go Report Card](https://goreportcard.com/badge/github.com/reportportal/landing-aggregator)](https://goreportcard.com/report/github.com/reportportal/landing-aggregator)


## Build

```
make help:
	@echo "vendor      - Install govendor and sync vendored dependencies"
	@echo "checkstyle  - executes bunch of checkstyle validators"
	@echo "fmt         - formats the project"
	@echo "test        - executes unit tests"
	@echo "build       - builds binary"
	@echo "docker      - Builds docker image"
	@echo "clean       - Cleans build-related files from working directory"
	@echo "release     - Builds docker container and pushes new version to DockerHUB"
```
```bash make docker ``` 
builds image with name 'reportportal/landing-info'
To create container execute 
 
```bash docker run --name landing-info -p 8080:8080 reportportal/landing-info```
 
## API
 
 Aggregator exposes two endpoints:
 
```/twitter```
Returns cache of tweets searched by provided in configuration hashtag
 
```/versions```
Returns latest versions of ReportPortal's Docker Images. Obtains this information from Docker HUB API

## Production deployment
Several instances of app are supposed to be deployed to provide fault-tolerance to distribute load.
There is Traefik which is reverse-proxy and load-balancer which does not require any service registry and allows zero-conf 
discovery via Docker API. Example can be found [here](docker-compose.yml)