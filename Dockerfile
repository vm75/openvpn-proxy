FROM alpine AS build

ARG DANTE_VERSION=1.4.3

RUN apk add --no-cache build-base
RUN wget https://www.inet.no/dante/files/dante-$DANTE_VERSION.tar.gz --output-document - | tar -xz && \
    cd dante-$DANTE_VERSION && \
    ac_cv_func_sched_setscheduler=no ./configure --disable-client && \
    make install

FROM alpine

COPY --from=build /usr/local/sbin/sockd /usr/local/sbin/sockd

RUN apk --no-cache update
RUN apk --no-cache upgrade
RUN apk --no-cache --no-progress add ip6tables iptables openvpn bind-tools tinyproxy inotify-tools

ARG IMAGE_VERSION
ARG BUILD_DATE

LABEL source="github.com/vm75/openvpn-proxy"
LABEL version="$IMAGE_VERSION"
LABEL created="$BUILD_DATE"

COPY usr /usr

ENV VPN_LOG_LEVEL=3 \
    KILL_SWITCH=on \
    HTTP_PROXY=on \
    SOCKS_PROXY=on \
    DEPENDENCIES=""

RUN mkdir -p /data
RUN addgroup root openvpn

VOLUME ["/data"]

# expose ports for http-proxy and socks-proxy
EXPOSE 8080/tcp 1080/tcp

HEALTHCHECK --interval=60s --timeout=15s --start-period=120s \
             CMD ls /data/var/openvpn-proxy.running

ENTRYPOINT [ "/usr/local/bin/openvpn-proxy" ]
