FROM --platform=${BUILDPLATFORM} golang:1.25.3-alpine3.22 AS builder

ENV APP_DIR=/go/src/github.com/org/repos

ARG TARGETOS=linux
ARG TARGETARCH=amd64

ADD . ${APP_DIR}
WORKDIR ${APP_DIR}

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
  -o app ./landinginfo.go

FROM --platform=${BUILDPLATFORM}  alpine:3.22
ENV APP_DIR=/go/src/github.com/org/repos
ARG APP_VERSION

LABEL authors="Andrei Varabyeu <andrei_varabyeu@epam.com>, Reingold Shekhtel <reingold_shekhtel@epam.com>"
LABEL version=${APP_VERSION}

RUN apk --no-cache add --upgrade apk-tools
COPY --from=builder ${APP_DIR}/app /opt/app
WORKDIR /opt

RUN adduser -D rpuser
USER rpuser

EXPOSE 8080
CMD ["./app"]
