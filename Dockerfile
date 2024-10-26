# Stage 1: Build sockd
FROM alpine AS build-sockd

# Set the working directory
WORKDIR /workdir

# Install build dependencies
RUN apk add --no-cache build-base

# Build dante from source
ARG DANTE_VERSION=1.4.3
RUN wget https://www.inet.no/dante/files/dante-$DANTE_VERSION.tar.gz --output-document - | tar -xz && \
    cd dante-$DANTE_VERSION && \
    ac_cv_func_sched_setscheduler=no ./configure --disable-client && \
    make install

# Stage 2: Build config-server
FROM golang:alpine AS build-config-server

# Set the working directory
WORKDIR /workdir

# Copy the server code
COPY server /workdir/server

# Build go server
# RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -C /workdir/server -ldflags="-s -w" -o /workdir/config-server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -C /workdir/server -o /workdir/config-server

# Stage 3: Create the final minimal image
FROM alpine

RUN apk --no-cache update
RUN apk --no-cache upgrade
RUN apk --no-cache --no-progress add ip6tables iptables openvpn bind-tools tinyproxy inotify-tools

ARG IMAGE_VERSION
ARG BUILD_DATE

LABEL source="github.com/vm75/openvpn-proxy"
LABEL version="$IMAGE_VERSION"
LABEL created="$BUILD_DATE"

# Copy binaries from build stage
COPY --from=build-sockd /usr/local/sbin/sockd /usr/local/sbin/sockd
COPY --from=build-config-server /workdir/config-server /opt/config-server/config-server

# Copy the server code
COPY usr /usr
COPY server/static /opt/config-server/static

ENV VPN_LOG_LEVEL=3 \
    KILL_SWITCH=on \
    HTTP_PROXY=on \
    SOCKS_PROXY=on \
    DEPENDENCIES=""

RUN mkdir -p /data
RUN addgroup root openvpn

VOLUME ["/data"]

# expose ports for http-proxy, socks-proxy and config-server
EXPOSE 8080/tcp 1080/tcp 80/tcp

HEALTHCHECK --interval=60s --timeout=15s --start-period=120s \
             CMD ls /data/var/openvpn-proxy.running

ENTRYPOINT [ "/usr/local/bin/openvpn-proxy" ]
