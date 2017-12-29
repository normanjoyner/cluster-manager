#!/bin/bash

SRC_DIR=$GOPATH/src/github.com/containership/cloud-agent
cd $SRC_DIR

vendor/k8s.io/code-generator/generate-groups.sh all \
  github.com/containership/cloud-agent/pkg/client github.com/containership/cloud-agent/pkg/apis \
  containership.io:v3
