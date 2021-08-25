#!/bin/sh

# exec 1>> /data/var/openvpn-proxy.log 2>&1

SELF=$(realpath "${0}")
SCRIPT_DIR=$(dirname "${SELF}")

TINY_PROXY_SCRIPT=${SCRIPT_DIR}/tinyproxy_wrapper.sh
SOCKS_PROXY_SCRIPT=${SCRIPT_DIR}/sockd_wrapper.sh
VPN_UP_SCRIPT=${SCRIPT_DIR}/vpn-up.sh
VPN_DOWN_SCRIPT=${SCRIPT_DIR}/vpn-down.sh

CONFIG_FILE=/data/config/vpn.ovpn
AUTH_FILE=/data/config/vpn.auth
LOG_FILE=/data/var/openvpn-proxy.log

shutting_down=no

function sigtermHandler {
    shutting_down=yes
    # When you run `docker stop` or any equivalent, a SIGTERM signal is sent to PID 1.

    if [ $monitor_handle ]; then
        echo -e "${task}: Stopping healthcheck script..." >> ${LOG_FILE}
        kill -TERM $monitor_handle
    fi

    if [ $openvpn_handle ]; then
        echo -e "${task}: Stopping OpenVPN..." >> ${LOG_FILE}
        kill -TERM $openvpn_handle
    fi

    sleep 1
    echo -e "${task}: Exiting." >> ${LOG_FILE}
    exit 0
}

function ensureVpnConfig() {
    config_file=/data/var/vpn.ovpn
    auth_file=/data/var/vpn.auth
    cp ${CONFIG_FILE} ${config_file}
    dos2unix ${config_file}
    cp ${AUTH_FILE} ${auth_file}
}

function setRoute() {
    if [ "$1" == "down" ] ; then
        echo -e "${task}: Add default gateway to route" >> ${LOG_FILE}
        ip route add default via ${default_gateway} dev eth0
    elif [ "$1" == "up" ] ; then
        echo -e "${task}: Remove default gateway from route" >> ${LOG_FILE}
        ip route del default via ${default_gateway} dev eth0
    fi

    echo -e "${task}: Flush iptables rules" >> ${LOG_FILE}
    iptables -F

    if [ "$1" == "down" ] ; then
        echo -e "${task}: Set route to DNS" >> ${LOG_FILE}
        iptables -A OUTPUT -o eth0 -p udp --dport 53 -j ACCEPT
        iptables -A OUTPUT -o eth0 -j REJECT
    else
        echo -e "${task}: Setting subnet routes" >> ${LOG_FILE}
        for subnet in ${SUBNETS//,/ }; do
            # create a route to it and...
            ip route add ${subnet} via ${default_gateway} dev eth0
            # allow connections
            iptables -A INPUT -s ${subnet} -j ACCEPT
        done
    fi

    echo -e "${task}: Set default routes" >> ${LOG_FILE}
    iptables -A INPUT -j ACCEPT -m state --state ESTABLISHED
    iptables -A INPUT -i lo -j ACCEPT
    iptables -A OUTPUT -o lo -j ACCEPT
}

function onVpnUp() {
    source /data/var/env

    ${TINY_PROXY_SCRIPT} &
    ${SOCKS_PROXY_SCRIPT} &

    setRoute up

    echo -e "${task}: Setting running script" >> ${LOG_FILE}
    touch /data/var/openvpn-proxy.running

    #ping -c 10 -A google.com
    #if [ $? -ne 0 ] ; then
    #	pkill openvpn
    #fi
}

function onVpnDown() {
    source /data/var/env

    echo -e "${task}: Stop socks proxy" >> ${LOG_FILE}
    pkill socks
    echo -e "${task}: Stop http proxy" >> ${LOG_FILE}
    pkill tinyproxy
    echo -e "${task}: Delete running script" >> ${LOG_FILE}
    rm -f /data/var/openvpn-proxy.running

    while : ; do
        echo -e "${task}: Reset routes" >> ${LOG_FILE}
        setRoute down

        server=$(grep '^remote ' /data/config/vpn.ovpn  | awk '{print $2}')
        nslookup $server &> /dev/null
        if [ $? -eq 0 ] ; then
            break
        fi
        echo -e "${task}: Unable to reach server. Trying again..." >> ${LOG_FILE}
        sleep 1
    done
}

function setEnv() {
    echo -e "${task}: Set env file:" >> ${LOG_FILE}
    env > /data/var/env

    default_gateway=$(ip r | grep 'default via' | cut -d " " -f 3)
    echo -e "default_gateway=${default_gateway}" >> /data/var/env
    echo /data/var/env
}

function monitor() {
    task="monitor"
    while : ; do
        # echo -e "${task}: Waiting 5 seconds" >> ${LOG_FILE}
        sleep 5
        if [ "$shutting_down" == "yes" ] ; then
            return
        fi
        # echo -e "${task}: Checking diffs" >> ${LOG_FILE}
        diff /data/config/vpn.ovpn /data/var/vpn.ovpn &> /dev/null
        result=$?
        if [ $result -eq 0 ] ; then
            diff /data/config/vpn.auth /data/var/vpn.auth &> /dev/null
            result=$?
        else
            echo -e "${task}: vpn config differs" >> ${LOG_FILE}
        fi
        if [ $result -eq 0 ] ; then
            ping -c 1 -A google.com
            result=$?
        else
            echo -e "${task}: vpn auth differs" >> ${LOG_FILE}
        fi
        if [ $result -ne 0 ] ; then
            echo -e "${task}: Killing openvpn" >> ${LOG_FILE}
        	kill -s INT ${openvpn_handle}
            exit
        fi
    done
}

function startVpn() {
    echo -e "${task}: Starting OpenVPN client" >> ${LOG_FILE}
    mkdir -p /data/var /data/config &> /dev/null

    echo -e "${task}: Clear logs" >> ${LOG_FILE}
    rm -f /data/var/*

    echo -e "${task}: Configure sigterm handler" >> ${LOG_FILE}
    trap sigtermHandler SIGTERM

    if [ ! -f ${CONFIG_FILE} -o ! -f ${AUTH_FILE} ] ; then
        echo -e "${task}: Invalid VPN config" >> ${LOG_FILE}
        exit 1
    fi

    setEnv

    setRoute init

    while [ "$shutting_down" == "no" ] ; do
        ensureVpnConfig

        echo -e "${task}: Call openvpn" >> ${LOG_FILE}
        openvpn --config ${config_file} \
        --auth-nocache \
        --connect-retry-max 10 \
        --auth-user-pass ${auth_file} \
        --status /data/var/openvpn.status 15 \
        --log /data/var/openvpn.log \
        --pull-filter ignore "route-ipv6" \
        --pull-filter ignore "ifconfig-ipv6" \
        --script-security 2 \
        --up-delay --up ${VPN_UP_SCRIPT} \
        --down ${VPN_DOWN_SCRIPT} \
        --up-restart \
        --ping-restart 10 \
        --group openvpn \
        --redirect-gateway autolocal \
        --cd /data/config &

        openvpn_handle=$!
        echo -e "${task}: openvpn_handle=${openvpn_handle}" >> ${LOG_FILE}

        monitor&
        monitor_handle=$!
        echo -e "${task}: monitor_handle=${monitor_handle}" >> ${LOG_FILE}

        wait $openvpn_handle

        echo -e "${task}: VPN down. $shutting_down" >> ${LOG_FILE}

        if [ "$shutting_down" == "no" ] ; then
            echo -e "${task}: Retrying openvpn connection ..." >> ${LOG_FILE}
            sleep 5;
        fi
    done

    echo -e "Done!"
}

function main() {
    case "$1" in
    --vpn-up)
        shift
        task="vpn-up"
        onVpnUp "$@"
        ;;
    --vpn-down)
        shift
        task="vpn-down"
        onVpnDown "$@"
        ;;
    *)
        task="openvpn-proxy"
        startVpn "$@"
        ;;
    esac
}

main "$@"
