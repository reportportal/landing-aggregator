# Landing Page Info Aggregator

[![Go Report Card](https://goreportcard.com/badge/github.com/reportportal/landing-aggregator)](https://goreportcard.com/report/github.com/reportportal/landing-aggregator)
[![Docker Pulls](https://img.shields.io/docker/pulls/reportportal/landing-aggregator.svg?maxAge=159200)](https://hub.docker.com/r/reportportal/landing-aggregator/)

[![Run in Postman](https://run.pstmn.io/button.svg)](https://app.getpostman.com/run-collection/39ce87bf716162454c2e)

Micro-service which serves next:

* cache tweets from https://twitter.com/ReportPortal_io
* return latest version numbers for ReportPortal services
* aggregates ReportPortal's GitHub statistics

## Build

```bash
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

Build image with name 'reportportal/landing-info':

```bash
make docker
```

To create container execute:

```bash
docker run --name landing-info -p 8080:8080 reportportal/landing-info
```

## API

Aggregator exposes endpoints:

```/```
Returns all the cached and aggregated data including tweets from Twitter and GitHub-related info

```/twitter```
Returns the feed cache from the Contentful CMS project as a Twitter-like feed. Includes only text fields.

```/versions```
Returns latest versions of ReportPortal's Docker Images. Obtains this information from GitHUB API

### Github aggregation details

```/github/contribution```
Returns commits and unique contributors for the last weeks
For example,

```json
{
  "commits": {
    "1": 0,
    "4": 201,
    "12": 1018
  },
  "unique_contributors": {
    "1": 20,
    "4": 52,
    "12": 80
  }
}
```

means that there were no commits for the current week, 201 commits for the last 4 weeks, etc.

```/github/stars```
Returns stars count for each repository and total count

```/github/issues```
Aggregates issue statistics from each organization repository

## Configuration

Aggregator can be configured through env variables. The following configuration options are available:

| ENV VAR                             |   Default Value    | Description                                   |
|-------------------------------------|:------------------:|-----------------------------------------------|
| PORT                                |        8080        | Application port                              |
| GITHUB_INCLUDE_BETA                 |       false        | Whether BETA versions should be included      |
| GITHUB_TOKEN                        |       false        | GitHUB API Token                              |
| GOOGLE_API_KEY                      |       false        | Google API Key                                |
| GOOGLE_PROJECT_ID                   |       false        | Google Cloud Project ID                       |
| GOOGLE_RECAPTCHA_KEY                |       false        | Google reCAPTCHA Site Key                     |
| GOOGLE_APPLICATION_CREDENTIALS      |       false        | Google Application Credentials JSON file path |
| GOOGLE_RECAPTCHA_SUBSCRIPTION_SCORE |        0.5         | reCAPTCHA minimum score for subscription form |
| YOUTUBE_BUFFER_SIZE                 |         10         | Number of videos to be cached                 |
| YOUTUBE_CHANNEL_ID                  |        Null        | YouTube channel ID                            |
| CONTENTFUL_TOKEN                    |        Null        | Contentful API Access Token                   |
| CONTENTFUL_SPACE_ID                 |    1n1nntnzoxp4    | Contentful Space ID                           |
| CONTENTFUL_LIMIT                    |         15         | Number of entries to be fetched and cached    |
| MAILCHIMP_API_KEY                   |        Null        | MailChimp API Key                             |
| MAILCHIMP_USER                      | landing-aggregator | MailChimp User                                |
| MAILCHIMP_TIMEOUT_SECONDS           |         3          | MailChimp Requests Timeout                    |

## Production deployment

Several instances of app should be deployed to provide fault-tolerance and distribute load.
There is [Traefik](traefik.io) which is reverse-proxy and load-balancer which does not require any
service registry and allows zero-conf discovery via Docker API.
Example can be found [here](docker-compose.yml)
