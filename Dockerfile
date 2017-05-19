FROM alpine:3.5

MAINTAINER Andrei Varabyeu <andrei_varabyeu@epam.com>

ADD scripts/ca-certificates.crt /etc/ssl/certs/

ADD ./bin/rpLandingInfo /

ENV PORT=8080

EXPOSE 8080
EXPOSE 6060
CMD ["/rpLandingInfo"]