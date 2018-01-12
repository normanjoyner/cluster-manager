# This dockerfile is only for Jenkins tests
FROM iron/go:dev

# add tools for debug and development purposes
RUN apk update && apk add vim && apk add iptables && apk add glide

ENV SRC_DIR=/gocode/src/github.com/containership/cloud-agent/

WORKDIR /app

# Add the source code:
ADD . $SRC_DIR

# Build it:
RUN cd $SRC_DIR && \
    glide install -v && \
    go build -o coordinator cmd/cloud_coordinator/coordinator.go && \
    cp coordinator /app/

ENTRYPOINT /app/coordinator
