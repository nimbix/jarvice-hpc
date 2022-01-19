#!/bin/bash

set -e

VERSION="XXX"
WORKDIR="pub"
INSTALL_PREFIX=${INSTALL_PREFIX:-/usr/local}
CLI_NAME="jarvice"
CLIENTS="CLIENTS"
GOOS="linux"
GOARCH="amd64"
RPMBUILD="rpmbuild/centos7"

function usage {
    cat <<EOF
Usage:
    $0 [options]

Options:
    --version <version>     Version to install                      (Default: $VERSION)
    --os                    Target os                               <linux | darwin | windows>
    --install-prefix        Path for installation                   (Default: $INTALL_PREFIX)

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
        --install-prefix)
            INSTALL_PREFIX=$2
            shift; shift
            ;;
        *)
            usage
            exit 1
            ;;
    esac
done

docker run --rm -ti $RPMBUILD bash -c "echo hello world &> /dev/null"

[ -d "pub" ] && WORKDIR=`TMPDIR=$PWD mktemp -d pub.XXX`
# noop if pub already exists
mkdir -p pub &> /dev/null

date > $WORKDIR/timestamp

[ ! -f "${CLIENTS}" ] && echo "missing ${CLIENTS} file" && exit 1

for client in `cat ${CLIENTS}`; do
    echo
    echo PACKAGING $client
    echo
    ./install.sh --os ${GOOS} --no-install --client $client --keep-cli --build --version ${VERSION}
    mv ${CLI_NAME}-${client} ${WORKDIR}
    cp ${client}/${client}-cli ${WORKDIR}
    RPM_NAME="jarvice-hpc-${client}-${VERSION//v}"
    mkdir -p ${RPM_NAME}/${INSTALL_PREFIX}/bin
    cp ${WORKDIR}/${CLI_NAME}-${client} ${RPM_NAME}/${INSTALL_PREFIX}/bin/${CLI_NAME}
    RETDIR=${PWD}
    source "${client}/${client}-cli"
    cd ${RPM_NAME}/${INSTALL_PREFIX}/bin
    for com in $COMS; do
        ln -s ${CLI_NAME} ${com}
    done
    cd ${RETDIR}
    tar -czvf ${RPM_NAME}.tar.gz ${RPM_NAME}/ &> /dev/null
    rm -rf ${RPM_NAME}/
done

cd ${WORKDIR}
PACKAGE_NAME="${CLI_NAME}_${VERSION}_${GOOS}_${GOARCH}.tar.gz"
tar -czvf ${PACKAGE_NAME} ${CLI_NAME}-* *-cli &> /dev/null
rm ${CLI_NAME}-* *-cli
cd ${RETDIR}

RPMARCH="x86_64"

for client in `cat ${CLIENTS}`; do
    RPM_NAME="jarvice-hpc-${client}-${VERSION//v}"
    # Create rpms
    cat <<EOF > jarvice-${client}.spec
Name:           jarvice-hpc-$client
Version:        ${VERSION//v}
Release:        1%{?dist}
Summary:        JARVICE HPC client for $client

Group:          Development/Tools
License:        BSD-2-Clause-Views
Source0:        ${RPM_NAME}.tar.gz
Prefix:         ${INSTALL_PREFIX}

%description
JARVICE HPC client

%prep
%setup -q


%build

%install
cp -rfa * %{buildroot}


%files
%dir ${INSTALL_PREFIX}
%dir ${INSTALL_PREFIX}/bin
${INSTALL_PREFIX}/bin/*


%changelog
EOF
    echo
    echo PACKAGING RPM FOR $client
    echo
    docker run -ti --rm -v "$PWD:/home/builder" \
        -w "/home/builder" \
        $RPMBUILD \
        /bin/bash -c "mkdir -p rpmbuild/SOURCES rpmbuild/SPECS \
        && cp jarvice-$client.spec rpmbuild/SPECS/ \
        && cp "${RPM_NAME}.tar.gz" rpmbuild/SOURCES/ \
        && rpmbuild --target ${RPMARCH} -bb rpmbuild/SPECS/jarvice-$client.spec &> /dev/null"
    cp rpmbuild/RPMS/${RPMARCH}/${RPM_NAME}*.rpm ${WORKDIR}/${RPM_NAME}.${RPMARCH}.rpm
    PACKAGE_NAME+=" ${RPM_NAME}.${RPMARCH}.rpm"
done

CHECKSUM="SHA256SUMS"

rm -rf rpmbuild/ *.spec *.tar.gz

cd ${WORKDIR}
sha256sum  ${PACKAGE_NAME} &> /dev/null > ${CHECKSUM} || shasum -a 256 ${PACKAGE_NAME} > ${CHECKSUM}

echo "PUBLISHED ${WORKDIR}"
