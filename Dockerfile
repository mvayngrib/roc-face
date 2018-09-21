FROM golang:1.11.0-stretch

WORKDIR /go/src/app

RUN go get -d github.com/gorilla/mux

COPY bin bin
COPY lib lib
COPY include include
COPY share share
COPY ROC.lic .
COPY go go

WORKDIR ./go

ENV LD_LIBRARY_PATH=/go/src/app/lib
ENV CGO_CFLAGS=-I/go/src/app/include
ENV CGO_LDFLAGS=-L/go/src/app/lib

EXPOSE 8080

CMD ["go", "run", "-v", "roc_server.go", "8080"]
