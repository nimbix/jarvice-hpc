#!/bin/bash

INSTALL_PREFIX=${INSTALL_PREFIX:-/usr/local/bin}

CLIENT=${1:-sge}
[ ! -d "$CLIENT" ] && echo "$CLIENT directory does not exists" && exit 1

source $CLIENT/$CLIENT-cli

rm -f ${INSTALL_PREFIX}/jarvice
for com in $COMS; do
    rm -f ${INSTALL_PREFIX}/$com
done

