# This dockerfile is only for Jenkins tests
FROM iron/go:1.9.3-dev

# add tools for debug and development purposes
RUN apk update && apk add vim && apk add iptables && apk add glide && apk add bash

ENV SRC_DIR=/gocode/src/github.com/containership/cloud-agent/
ENV GOPATH=/gocode

WORKDIR /app

# Add the source code:
ADD . $SRC_DIR

# Build it:
RUN cd $SRC_DIR && \
    glide install -v && \
    go build -o coordinator cmd/cloud_coordinator/coordinator.go && \
    cp coordinator /app/

ENTRYPOINT /app/coordinator

# TODO glog wants to log to a file by default. Don't use glog. See issue #36.
CMD ["-logtostderr=true"]
