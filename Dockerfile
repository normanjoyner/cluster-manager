# This dockerfile is only for Jenkins tests
FROM golang:alpine

RUN apk add --no-cache git gcc musl-dev
# add tools for debug and development purposes
RUN apk update && apk add vim && apk add iptables && apk add glide

WORKDIR /app

# Add the source code:
ADD . /go/src/github.com/containership/cloud-agent/

# Build it:
RUN cd /go/src/github.com/containership/cloud-agent/ && \
    glide install && \
    go build -o coordinator cmd/cloud_coordinator/coordinator.go && \
    cp coordinator /app/

ENTRYPOINT /app/coordinator
