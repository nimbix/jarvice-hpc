#!/bin/bash

set -e

VERSION="XXX"
WORKDIR="pub"
CLI_NAME="jarvice"
CLIENTS="CLIENTS"
GOOS="linux"
GOARCH="amd64"

function usage {
    cat <<EOF
Usage:
    $0 [options]

Options:
    --version <version>     Version to install                      (Default: $VERSION)
    --os                    Target os                               <linux | darwin | windows>
    --debug                 Enable debug logging

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

[ -d "pub" ] && WORKDIR=`TMPDIR=$PWD mktemp -d pub.XXX`
# noop if pub already exists
mkdir -p pub &> /dev/null

date > $WORKDIR/timestamp

[ ! -f "${CLIENTS}" ] && echo "missing ${CLIENTS} file" && exit 1

for client in `cat ${CLIENTS}`; do
    echo
    echo PACKAGING $client
    echo
    ./install.sh --no-install --client $client --keep-cli --build
    mv ${CLI_NAME}-${client} ${WORKDIR}
    cp ${client}/${client}-cli ${WORKDIR}
done

cd ${WORKDIR}
PACKAGE_NAME="${CLI_NAME}_${VERSION}_${GOOS}_${GOARCH}.tar.gz"
tar -czvf ${PACKAGE_NAME} ${CLI_NAME}-* *-cli
CHECKSUM="SHA256SUMS"
rm ${CLI_NAME}-* *-cli

sha256sum  ${PACKAGE_NAME} &> /dev/null > ${CHECKSUM} || shasum -a 256 ${PACKAGE_NAME} > ${CHECKSUM}

echo "PUBLISHED ${WORKDIR}"
