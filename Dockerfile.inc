FROM vm75/openvpn-proxy

COPY usr /usr

HEALTHCHECK --interval=60s --timeout=15s --start-period=120s \
             CMD ls /data/var/openvpn-proxy.running

ENTRYPOINT [ "/usr/local/bin/openvpn-proxy.sh" ]
