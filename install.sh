#!/bin/bash

set -e

VERSION=v0.0.2
CLIENT=sge
INSTALL_PREFIX=${INSTALL_PREFIX:-/usr/local/bin}
GHPATH="https://github.com/nimbix/jarvice-hpc/releases/download"
GOOS="linux"
GOARCH="amd64"
CLI_NAME="jarvice"
DEBUG="&> /dev/null"

function usage {
    cat <<EOF
Usage:
    $0 [options]

Options:
    --version <version>     Version to install                      (Default: $VERSION)
    --client  <client>      HPC client to install                   (Default: $CLIENT)
    --build                 Build client from source
    --keep-cli              Keep CLI binary used for installation
    --install-prefix        Path for installation                   (Default: $INSTALL_PREFIX)
    --os                    Target os                               <linux | darwin | windows>
    --debug                 Enable debug logging
    --no-install            Skip install

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
        --keep-cli)
            KEEP_CLI=true
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
        --debug)
            unset DEBUG
            shift
            ;;
        --no-install)
            NOINSTALL=true
            shift
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

    GO_BUILD_OPT="-v -o ${CLI_NAME}-${CLIENT} -a -ldflags '-extldflags -static -s -w'"

    docker run -ti --rm -v "$PWD":/usr/src/jarvice-hpc \
        -w /usr/src/jarvice-hpc \
        -e GOOS=${GOOS} \
        -e CGO_ENABLED=0 \
        golang:1.14 \
        /bin/bash -c "go get github.com/jessevdk/go-flags \
        && mkdir -p /go/src/jarvice.io \
        && ln -s /usr/src/jarvice-hpc/core /go/src/jarvice.io \
        && go build $GO_BUILD_OPT $CLIENT/*.go ${DEBUG}"

    echo "BUILD COMPLETE"

    for file in `ls *.go`; do
        rm -f $CLIENT/$file
    done
else
    command -v wget &> /dev/null || (echo "wget missing" && exit 1)

    WD=`pwd`
    WORKDIR=`mktemp -d`
    cd "${WORKDIR}"
    wget "${GHPATH}/${VERSION}/SHA256SUMS" ${DEBUG} \
        || (echo "$VERSION SHA256SUMS Not Found" && exit 1)
    wget "${GHPATH}/${VERSION}/${CLI_NAME}_${VERSION}_${GOOS}_${GOARCH}.tar.gz" ${DEBUG} \
        || (echo "$VERSION archive Not Found" && exit 1)
    CHECKSUM="-c SHA256SUMS"
    sha256sum ${CHECKSUM} &> /dev/null || shasum -a 256 ${CHECKSUM} &> /dev/null \
        || (echo "Checksum failed" && exit 1)
    tar -xvf "${CLI_NAME}_${VERSION}_${GOOS}_${GOARCH}.tar.gz" ${DEBUG}

    [ -e "$CLIENT-cli" ] || (echo "Missing $CLIENT-cli" && exit 1)
    source "$CLIENT-cli"

    cp ${CLI_NAME}-* $WD
    cd $WD
fi

[ ! "$BUILD" ] && rm -r "${WORKDIR}"

[ "${NOINSTALL}" ] && exit 0

if [ -z "$COMS" ]; then
    usage
    exit 1
fi

mkdir -p ${INSTALL_PREFIX}

echo "INSTALLING $CLIENT TO $INSTALL_PREFIX"

cp ${CLI_NAME}-${CLIENT} ${INSTALL_PREFIX}/${CLI_NAME}
for com in $COMS; do
    ln -fs ${INSTALL_PREFIX}/${CLI_NAME} ${INSTALL_PREFIX}/$com
done
[ ! "$KEEP_CLI" ] && rm ${CLI_NAME}-*
echo "INSTALLATION COMPLETE"
