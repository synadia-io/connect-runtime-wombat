FROM golang:1.24.6 AS build

ENV CGO_ENABLED=0
ENV GOOS=linux
RUN useradd -u 10001 wombat
RUN sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin

ARG VERSION=dev
ARG COMMIT_HASH=unknown
ARG BUILD_DATE=unknown

ENV VERSION=${VERSION}
ENV COMMIT_HASH=${COMMIT_HASH}
ENV BUILD_DATE=${BUILD_DATE}

WORKDIR /go/src/github.com/synadia-io/connect-runtime-wombat/
# Update dependencies: On unchanged dependencies, cached layer will be reused
COPY . /go/src/github.com/synadia-io/connect-runtime-wombat/
RUN go mod tidy

# Build
# Tag timetzdata required for busybox base image:
# https://github.com/benthosdev/benthos/issues/897
RUN task build TAGS="timetzdata"

# Pack
FROM busybox AS package

LABEL maintainer="Synadia <code@synadia.com>"
LABEL org.opencontainers.image.source="https://github.com/synadia-io/connect-runtime-wombat"

WORKDIR /

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /go/src/github.com/synadia-io/connect-runtime-wombat/target/connect-runtime-wombat .

USER wombat

EXPOSE 4195

ENTRYPOINT ["/connect-runtime-wombat"]
