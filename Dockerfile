FROM alpine:3.5

MAINTAINER Andrei Varabyeu <andrei_varabyeu@epam.com>

RUN apk --no-cache add ca-certificates

RUN adduser -D rpuser
USER rpuser

ADD ./bin/rpLandingInfo /

ENV PORT=8080

EXPOSE 8080
CMD ["/rpLandingInfo"]
