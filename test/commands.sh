#!/bin/sh

TEST_DIR=/src/test
VAR_DIR=${TEST_DIR}/var
VPN_UP_SCRIPT=${TEST_DIR}/vpn-up
VPN_DOWN_SCRIPT=${TEST_DIR}/vpn-down

setup() {
    mkdir -p ${VAR_DIR}
    ln -s ${TEST_DIR}/commands.sh /bin/cmd
    apk --no-cache update
    apk --no-cache upgrade
    apk --no-cache --no-progress add ip6tables iptables openvpn bind-tools inotify-tools curl build-base
    server
}

server() {
    while true; do
        {
            echo -e 'HTTP/1.1 200 OK\r\n'
            echo "Port = 80";
            echo "WAN IP = $(curl -s ifconfig.me)";
            echo "Date = $(date)";
        } | nc -l -p 80 &> /dev/null
    done &
    while true; do
        {
            echo -e 'HTTP/1.1 200 OK\r\n'
            echo "Port = 81";
            echo "WAN IP = $(curl -s ifconfig.me)";
            echo "Date = $(date)";
        } | nc -l -p 81 &> /dev/null
    done &
}

run_openvpn() {
    VPN_CONF=${VAR_DIR}/vpn.ovpn
    VPN_AUTH=${VAR_DIR}/vpn.auth
    RETRY_INTERVAL=30
    OPENVPNPROXY_LOG=${VAR_DIR}/openvpn-proxy.log
    openvpn \
        --client \
        --cd ${VAR_DIR} \
        --config ${VPN_CONF} \
        --auth-user-pass ${VPN_AUTH} \
        --auth-nocache \
        --verb ${VPN_LOG_LEVEL:-3} \
        --log ${VAR_DIR}/openvpn.log \
        --status ${VAR_DIR}/openvpn.status ${RETRY_INTERVAL} \
        --ping-restart ${RETRY_INTERVAL} \
        --connect-retry-max 3 \
        --script-security 2 \
        --up ${VAR_DIR}/openvpn-proxy --up-delay \
        --down ${VAR_DIR}/openvpn-proxy \
        --up-restart \
        --pull-filter ignore route-ipv6 \
        --pull-filter ignore ifconfig-ipv6 \
        --pull-filter ignore block-outside-dns \
        --redirect-gateway def1 \
        --remote-cert-tls server \
        --data-ciphers AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305:AES-256-CBC:AES-128-CBC \
        >> ${OPENVPNPROXY_LOG} 2>&1 &
}

main() {
    case "$1" in
        setup)
            setup
            ;;
        vpn)
            run_openvpn
            ;;
        res)
            cp -a /etc/resolv.conf /etc/resolv.conf.bak
            ;;
        build)
            cd /src/server
            CGO_ENABLED=1 go build -o ${VAR_DIR}/openvpn-proxy .
            ;;
        serve)
            server
            ;;
        reset)
            ${VPN_DOWN_SCRIPT} ; pkill -9 openvpn ; cmd build ; cmd vpn
            ;;
        stop)
            pkill -15 openvpn
            ;;
        *)
            echo "Unknown command: $1"
            exit 1
            ;;
    esac
}

main "$@"