#!/bin/bash

RANKONE_TAR=$1
RANKONE_LICENSE=$2

curl "$RANKONE_TAR" > rankone.tar.gz
tar -xzvf rankone.tar.gz
mv roc-linux-x64-fma3 rankone
cp $RANKONE_LICENSE rankone/
curl https://raw.githubusercontent.com/mvayngrib/roc-face/master/go/roc_server.go > rankone/go/roc_server.go
curl https://raw.githubusercontent.com/mvayngrib/roc-face/master/go/serve.sh > rankone/go/serve.sh
chmod +x rankone/go/serve.sh
cd rankone/go
go get github.com/gorilla/mux

echo "
to start the server:
./serve.sh [PORT]
"
