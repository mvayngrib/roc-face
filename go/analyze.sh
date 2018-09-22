#!/bin/bash
export DYLD_LIBRARY_PATH=../lib
export CGO_CFLAGS=-I../include
export CGO_LDFLAGS=-L../lib

go run -v roc_server.go analyze "$1"
