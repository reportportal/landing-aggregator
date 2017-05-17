# Landing Page Info Aggregator

[![Build Status](https://travis-ci.org/reportportal/landing-aggregator.svg?branch=master)](https://travis-ci.org/reportportal/landing-aggregator)
[![Go Report Card](https://goreportcard.com/badge/github.com/reportportal/landing-aggregator)](https://goreportcard.com/report/github.com/reportportal/landing-aggregator)

Micro-service which serves next:
* cache tweets from https://twitter.com/ReportPortal_io
* return latest version numbers for ReportPortal services

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

### Twitter aggregation details
Aggregator supports two modes:
* 'Hashtag' or streaming mode. Enabled by default. Buffers all tweets found by provided search term.
Uses Twitter Streaming API to keep buffer up to date
* 'Follow user' mode. Enabled by providing search term starting from '@'. Buffers all messages of specified user except retweets and replies.
Uses long-pooling to keep buffer up to date.
 
```/versions```
Returns latest versions of ReportPortal's Docker Images. Obtains this information from GitHUB API

## Configuration
Aggregator can be configured through env variables. The following configuration options are available:

| ENV VAR                       | Default Value    | Required    | Description                  |
| ------------------------------|:----------------:| -----------:|-----------------------------:|
| PORT                          | 8080             | false       |Application port              |
| TWITTER_CONSUMER              |                  | true        |Twitter API consumer key      |
| TWITTER_CONSUMER_SECRET       |                  | true        |Twitter API consumer secret   |
| TWITTER_TOKEN                 |                  | true        |Twitter API token             |
| TWITTER_TOKEN_SECRET          |                  | true        |Twitter API token  secret|
| TWITTER_BUFFER_SIZE|10|false|Tweets buffer size|
| TWITTER_SEARCH_TERM|@reportportal_io|false|Tweets search term|
| GITHUB_INCLUDE_BETA|false|false|Whether BETA versions should be included|
| GITHUB_TOKEN||false|GitHUB API Token| 

## Production deployment
Several instances of app are supposed to be deployed to provide fault-tolerance to distribute load.
There is Traefik which is reverse-proxy and load-balancer which does not require any service registry and allows zero-conf 
discovery via Docker API. Example can be found [here](docker-compose.yml)
