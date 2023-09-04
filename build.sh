#!/bin/bash

DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
REVISION=$(git rev-list --count $(git branch --show-current))
VERSION=v1.${REVISION}

sudo docker build --build-arg BUILD_DATE="${DATE}" --build-arg IMAGE_VERSION=${VERSION} --format docker -t vm75/openvpn-proxy .

IMAGE_ID=$(sudo docker images | grep openvpn-proxy | grep latest | awk '{ print $3}')

echo "tagging with ${IMAGE_ID} vm75/openvpn-proxy:latest vm75/openvpn-proxy:${VERSION}"

sudo docker tag ${IMAGE_ID} vm75/openvpn-proxy:latest vm75/openvpn-proxy:${VERSION}

if [ "$2" == push ] ; then
	sudo docker login
	if [[ $? -eq 0 ]] ; then
		sudo docker push vm75/openvpn-proxy
	fi
fi
