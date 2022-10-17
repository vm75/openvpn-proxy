FROM alpine AS build

RUN apk add --no-cache build-base openssl-libs-static openssl-dev
COPY src /src
RUN cd /src && \
    make
