FROM alpine

# Build-time metadata as defined at http://label-schema.org
ARG BUILD_DATE
ARG VCS_REF
ARG VERSION
LABEL org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.name="vm75/openvpn-proxy" \
      org.label-schema.description="An Alpine-based OpenVPN Client with Proxy." \
      org.label-schema.url="https://hub.docker.com/r/vm75/openvpn-proxy/" \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url="https://github.com/vm75/openvpn-proxy" \
      org.label-schema.vendor="Vijay Mohan" \
      org.label-schema.version=$VERSION \
      org.label-schema.schema-version="1.0"

ENV VPN_LOG_LEVEL=3 \
    KILL_SWITCH=on \
    HTTP_PROXY=on \
    SOCKS_PROXY=on

COPY build/sockd /usr/local/sbin/sockd
COPY usr /usr

RUN apk --no-cache --purge -uU --no-progress add ip6tables iptables openvpn bind-tools tinyproxy && \
    chmod u+s /sbin/ip /bin/chmod /usr/sbin/openvpn /usr/local/sbin/sockd /usr/bin/tinyproxy && \
    mkdir -p /data
# RUN addgroup root openvpn

VOLUME ["/data"]

HEALTHCHECK --interval=60s --timeout=15s --start-period=120s \
    CMD ip a show dev tun0

ENTRYPOINT [ "/usr/local/bin/run-openvpn-proxy.sh" ]
