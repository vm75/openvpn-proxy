#!/bin/bash

SELF=$(realpath "${0}")
SCRIPT_DIR=$(dirname "${SELF}")

usage() {
    echo "usage: ${SELF} [--test] [--push] [--socks-version <ver>] [--version <ver>]"
}

buildSockd() {
    dante_version=$1
    if [[ ! -d build ]] ; then
        mkdir build
    fi
    if [[ ! -f build/sockd ]] ; then
        sudo docker build --file Dockerfile.sockd --build-arg DANTE_VERSION=${dante_version} -v ${PWD}/build:/build .
    fi
}

build() {
    image_name=$1
    version=$2

    sudo docker build \
        --build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
        --build-arg VCS_REF=`git rev-parse --short HEAD` \
        --build-arg VERSION=${version} \
        --format docker --tag ${image_name} .

    IMAGE_ID=$(sudo docker images | grep ${image_name} | grep latest | awk '{ print $3}')

    echo "tagging with ${IMAGE_ID} ${image_name}:latest ${image_name}:${version}"

    sudo docker tag ${IMAGE_ID} ${image_name}:latest ${image_name}:${version}
}

main() {
    cd ${SCRIPT_DIR}

    revision=$(git rev-list --count $(git branch --show-current))
    version=v1.${revision}
    dante_version=1.4.3
    push=0
    test=""
    image_name=vm75/openvpn-proxy

    shortopts="i:v:d:tp"
    longopts="image-name:,version:,dante-version:,test,push"
    argv=$(getopt -o "$shortopts" -l "$longopts" -n $(basename $0) -- "$@")
    if [ $? -ne 0 ] ; then
        usage
    fi
    eval set -- "$argv"

    # extract options and their arguments into variables.
    while true ; do
        local opt=$1 ; shift
        case "$opt" in
        -t|--test)
            test="-test"
            ;;
        -v|--version)
            version="$1" ; shift
            ;;
        -s|--dante-version)
            dante_version="$1" ; shift
            ;;
        -i|--image-name)
            image_name="$1" ; shift
            ;;
        -p|--push)
            push=1
            ;;
        -h|--help)
            usage
            ;;
        --)
            break
            ;;
        *)
            usage
            ;;
        esac
    done

    buildSockd ${dante_version}

    image_name=${image_name}${test}
    build ${image_name} ${version}

    if [[ ${push} -eq 1 && "${test}" == "" ]] ; then
        sudo docker login
        if [[ $? -eq 0 ]] ; then
            sudo docker push ${image_name}
        fi
    fi
}

main "$@"
