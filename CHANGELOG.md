# Changelog

## 1.0
##### Released: 13 March 2017

* Initial DockerHub Release 

## 1.1
##### Released: 16 March 2017

### New Features

* 404/Not Found handler
* Low memory consumption by caching only needed tweet info
* add changelog

### Bugfixes

* build script: Release to DockerHUB prior to making git tag

## v1.5
##### Released: 31 March 2017

### New Features

* 'INCLUDE_BETA' env variable to configre whether BETA versions should be included in the list 
* Twitter data now contains entities and extended_entities

## v1.18
##### Released: 17 May 2017

### New Features

*  Replace streaming with long-polling for 'follow' mode

## v1.19
##### Released: 17 May 2017

### Bugfixes

*  Do not add count to the requests into 'follow' mode since twitter includes retweets in this case.
 See [the docs](https://dev.twitter.com/rest/reference/get/statuses/user_timeline)
 
## v1.20
##### Released: 17 May 2017

### Bugfixes

*  Traverse result in reverse order to make newest tweets at the bottom of the buffer 

## v1.21
##### Released: 17 May 2017

### New Features

*  Enable extended tweet mode
 
## v1.22
##### Released: 18 May 2017
### Bugixes
*  Fix presence of replies in 'follow' user mode 

## v1.24
##### Released: 23 May 2017
### New Features
*  Cache stars count for each org repository

## v1.27
##### Released: 29 May 2017
### New Features
* GitHub statistics aggregation
* Composite root endpoint for all aggregated data


## v1.28
##### Released: 8 June 2017
### Bugfixes
* Only 30 first organization repositories were processed by aggregator

## v1.31
##### Released: 10 June 2017
### Bugfixes
* Remove profiler

## v1.32
##### Released: 8 Aug 2017
### Bugfixes
* Add retweets to tweets cache

## v1.34
##### Released: 8 Aug 2017
### Bugfixes
* Improved dockerfile
