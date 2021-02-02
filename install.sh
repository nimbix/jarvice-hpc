#!/bin/bash

set -e

INSTALL_PREFIX=${INSTALL_PREFIX:-/usr/local/bin}

CLIENT=${1:-sge}
[ ! -d "$CLIENT" ] && echo "$CLIENT directory does not exists" && exit 1

source $CLIENT/$CLIENT-cli

cp *.go $CLIENT

docker run -ti --rm -v "$PWD":/usr/src/jarvice-hpc \
    -w /usr/src/jarvice-hpc \
    -e GOOS=linux \
    -e CGO_ENABLED=0 \
    golang:1.14 \
    /bin/bash -c "go get github.com/jessevdk/go-flags \
    && mkdir -p /go/src/jarvice.io \
    && ln -s /usr/src/jarvice-hpc/core /go/src/jarvice.io \
    && go build -v -o jarvice -a -ldflags '-extldflags -static -s -w' $CLIENT/*.go"

for file in `ls *.go`; do
    rm -f $CLIENT/$file
done

mkdir -p ${INSTALL_PREFIX}
mv jarvice ${INSTALL_PREFIX}
for com in $COMS; do
    ln -fs ${INSTALL_PREFIX}/jarvice ${INSTALL_PREFIX}/$com
done
