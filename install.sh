#!/bin/bash

set -e

VERSION=v0.0.2
CLIENT=sge
INSTALL_PREFIX=${INSTALL_PREFIX:-/usr/local/bin}
GHPATH="https://github.com/nimbix/jarvice-hpc/releases/download"
GOOS="linux"

function usage {
    cat <<EOF
Usage:
    $0 [options]

Options:
    --version <version>     Version to install      (Default: $VERSION)
    --client  <client>      HPC client to install   (Default: $CLIENT)
    --build                 Build client from source
    --install-prefix        Path for installation   (Default: $INSTALL_PREFIX)
    --os                    Target os               <linux | darwin | windows>

Example:
    $0 --version $VERSION
EOF
}

while [ $# -gt 0 ]; do
    case $1 in
        --help)
            usage
            exit 0
            ;;
        --version)
            VERSION=$2
            shift; shift
            ;;
        --client)
            CLIENT=$2
            shift; shift
            ;;
        --build)
            BUILD=true
            shift
            ;;
        --install-prefix)
            INSTALL_PREFIX=$2
            shift; shift
            ;;
        --os)
            GOOS=$2
            shift; shift
            ;;
        *)
            usage
            exit 1
            ;;
    esac
done

if [ "$BUILD" = true ]; then

    [ ! -d "$CLIENT" ] && echo "$CLIENT directory does not exists" && exit 1

    command -v docker &> /dev/null || (echo "docker missing" && exit 1)

    source $CLIENT/$CLIENT-cli

    cp *.go $CLIENT

    rm $CLIENT/template.go || true

    echo "BUILDING $CLIENT FOR $GOOS"

    GO_BUILD_OPT="-v -o jarvice -a -ldflags '-extldflags -static -s -w'"

    docker run -ti --rm -v "$PWD":/usr/src/jarvice-hpc \
        -w /usr/src/jarvice-hpc \
        -e GOOS=${GOOS} \
        -e CGO_ENABLED=0 \
        golang:1.14 \
        /bin/bash -c "go get github.com/jessevdk/go-flags \
        && mkdir -p /go/src/jarvice.io \
        && ln -s /usr/src/jarvice-hpc/core /go/src/jarvice.io \
        && go build $GO_BUILD_OPT $CLIENT/*.go"

    for file in `ls *.go`; do
        rm -f $CLIENT/$file
    done
else
    command -v wget &> /dev/null || (echo "wget missing" && exit 1)

    WD=`pwd`
    WORKDIR=`mktemp -d`
    cd "${WORKDIR}"
    wget "${GHPATH}/${VERSION}/SHA256SUMS" &> /dev/null \
        || (echo "$VERSION SHA256SUMS Not Found" && exit 1)
    wget "${GHPATH}/${VERSION}/jarvice_linux_amd64.tar.gz" &> /dev/null \
        || (echo "$VERSION archine Not Found" && exit 1)
    CHECKSUM="-c SHA256SUMS"
    sha256sum ${CHECKSUM} &> /dev/null || shasum -a 256 ${CHECKSUM} &> /dev/null \
        || (echo "Checksum failed" && exit 1)
    tar -xvf "jarvice_linux_amd64.tar.gz" &> /dev/null

    [ -e "$WD/$CLIENT-cli" ] || (echo "Missing $CLIENT-cli" && exit 1)
    source "$WD/$CLIENT-cli"

    cp jarvice $WD
    cd $WD
fi

if [ -z "$COMS" ]; then
    usage
    exit 1
fi

mkdir -p ${INSTALL_PREFIX}

echo "INSTALLING $CLIENT TO $INSTALL_PREFIX"

mv jarvice ${INSTALL_PREFIX}
for com in $COMS; do
    ln -fs ${INSTALL_PREFIX}/jarvice ${INSTALL_PREFIX}/$com
done
[ ! "$BUILD" ] && rm -r "${WORKDIR}"
echo "INSTALLATION COMPLETE"
