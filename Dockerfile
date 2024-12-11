FROM golang:1.23.2 AS build

ENV CGO_ENABLED=0
ENV GOOS=linux
RUN useradd -u 10001 wombat
RUN sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin

WORKDIR /go/src/github.com/synadia-labs/vent/
# Update dependencies: On unchanged dependencies, cached layer will be reused
COPY . /go/src/github.com/synadia-labs/vent/
RUN go mod tidy

# Build
# Tag timetzdata required for busybox base image:
# https://github.com/benthosdev/benthos/issues/897
RUN task runtimes:wombat:build TAGS="timetzdata"

# Pack
FROM busybox AS package

LABEL maintainer="Synadia <code@synadia.com>"
LABEL org.opencontainers.image.source="https://github.com/synadia-labs/vent"

WORKDIR /

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /go/src/github.com/synadia-labs/vent/target/runtime-wombat .

USER wombat

EXPOSE 4195

ENTRYPOINT ["/runtime-wombat"]
