FROM alpine:3.17

MAINTAINER Andrei Varabyeu <andrei_varabyeu@epam.com>

RUN apk --no-cache add ca-certificates

ADD ./bin/landinginfo /opt/landinginfo
WORKDIR /opt

RUN adduser -D rpuser
USER rpuser

ENV PORT=8080

EXPOSE 8080
CMD ["./landinginfo"]
