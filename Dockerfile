# Stage 1: Build dante sockd and openvpn-proxy
FROM golang:alpine AS build

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

# Copy the server code
COPY server /workdir/server

# Build go server
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -C /workdir/server -ldflags="-s -w" -o /workdir/openvpn-proxy

# Stage 2: Create the final minimal image
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
COPY --from=build /usr/local/sbin/sockd /usr/local/sbin/sockd
COPY --from=build /workdir/openvpn-proxy /opt/openvpn-proxy/openvpn-proxy

# Copy the server code
COPY usr /usr
COPY server/static /opt/openvpn-proxy/static

ENV VPN_LOG_LEVEL=3 \
    KILL_SWITCH=on \
    HTTP_PROXY=on \
    SOCKS_PROXY=on

RUN mkdir -p /data
RUN addgroup root openvpn

VOLUME ["/data"]

# expose ports for http-proxy, socks-proxy and openvpn-proxy
EXPOSE 8080/tcp 1080/tcp 80/tcp

HEALTHCHECK --interval=60s --timeout=15s --start-period=120s \
    CMD ls /data/var/openvpn.pid

ENTRYPOINT [ "/opt/openvpn-proxy/openvpn-proxy" ]
