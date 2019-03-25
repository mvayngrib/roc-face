#!/bin/bash

HERE=$(dirname $0)

export DYLD_LIBRARY_PATH=$(realpath "$HERE/../lib")
export LD_LIBRARY_PATH=$(realpath "$HERE/../lib")
export CGO_CFLAGS=-I$(realpath "$HERE/../include")
export CGO_LDFLAGS=-L$(realpath "$HERE/../lib")

PORT=${1-10001}

go run -v $(dirname $0)/roc_server.go serve $PORT
