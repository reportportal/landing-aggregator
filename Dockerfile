FROM alpine:3.5

MAINTAINER Andrei Varabyeu <andrei_varabyeu@epam.com>

RUN apk --no-cache add ca-certificates

ADD ./bin/rplandinginfo /opt/rplandinginfo
WORKDIR /opt

RUN adduser -D rpuser
USER rpuser

ENV PORT=8080

EXPOSE 8080
CMD ["./rplandinginfo"]
