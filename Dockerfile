FROM alpine:3.5

MAINTAINER Andrei Varabyeu <andrei_varabyeu@epam.com>

RUN apk add --no-cache ca-certificates

ADD ./bin/rpLandingInfo /

EXPOSE 8080
ENTRYPOINT ["/rpLandingInfo"]