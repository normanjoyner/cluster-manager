# This dockerfile is only for Jenkins tests
FROM iron/go:1.9.3-dev

# add tools for debug and development purposes
RUN apk update && apk add glide && apk add bash

ENV SRC_DIR=/gocode/src/github.com/containership/cloud-agent/
ENV GOPATH=/gocode

WORKDIR /app

# Glide install before adding rest of source so we can cache the resulting
# vendor dir
ADD glide.yaml glide.lock $SRC_DIR
RUN cd $SRC_DIR && \
        glide install -v

# Add the source code:
ADD . $SRC_DIR

WORKDIR $SRC_DIR
