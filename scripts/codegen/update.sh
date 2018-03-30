#!/bin/bash

SCRIPT_NAME=$(basename $0)

SRC_DIR=$GOPATH/src/github.com/containership/cloud-agent

go get -u k8s.io/gengo
rm -rf ./vendor/github.com/golang/glog
rm -rf ./vendor/github.com/spf13/pflag

cd $SRC_DIR

vendor/k8s.io/code-generator/generate-groups.sh all \
  github.com/containership/cloud-agent/pkg/client github.com/containership/cloud-agent/pkg/apis \
  "containership.io:v3 provision.containership.io:v3"
