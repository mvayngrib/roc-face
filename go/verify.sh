#!/bin/bash
export DYLD_LIBRARY_PATH=../lib
export CGO_CFLAGS=-I../include
export CGO_LDFLAGS=-L../lib

go run -v roc_example_verify.go ../data/josh_1.jpg ../data/josh_2.jpg