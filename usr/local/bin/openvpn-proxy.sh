#!/bin/sh

# exec 1>> /data/var/openvpn-proxy.log 2>&1

SELF=$(realpath "${0}")
SCRIPT_DIR=$(dirname "${SELF}")
ETC_DIR=/usr/local/etc
CONFIG_DIR=/data/config
VAR_DIR=/data/var

# external scripts
VPN_UP_SCRIPT=${SCRIPT_DIR}/vpn-up.sh
VPN_DOWN_SCRIPT=${SCRIPT_DIR}/vpn-down.sh

# config files
VPN_CONF=vpn.ovpn
VPN_AUTH=vpn.auth
TINYPROXY_CONF=tinyproxy.conf
SOCKD_CONF=sockd.conf

# log files
OPENVPNPROXY_LOG=openvpn-proxy.log
TINYPROXY_LOG=tinyproxy.log
SOCKD_LOG=sockd.log

RETRY_INTERVAL=5

function log() {
    date +"%D %R] ${task}: $@" >> ${VAR_DIR}/${OPENVPNPROXY_LOG}
}

shutting_down=no
function sigtermHandler {
    shutting_down=yes
    # When you run `docker stop` or any equivalent, a SIGTERM signal is sent to PID 1.

    if [ $monitor_pid ]; then
        log "Stopping healthcheck script..."
        kill -TERM $monitor_pid
    fi

    if [ $openvpn_pid ]; then
        log "Stopping OpenVPN..."
        kill -TERM $openvpn_pid
    fi

    sleep 1
    exit 0
}

function saveVpnConfig() {
    log "Copying vpn config to ${VAR_DIR}"
    cp ${CONFIG_DIR}/${VPN_CONF} ${VAR_DIR}/${VPN_CONF}
    dos2unix ${VAR_DIR}/${VPN_CONF}
    cp ${CONFIG_DIR}/${VPN_AUTH} ${VAR_DIR}/${VPN_AUTH}
    dos2unix ${VAR_DIR}/${VPN_AUTH}
}

function configureRoutes() {
    log "Configure routes for vpn-$1"

    if [ "$1" == "down" ] ; then
        log "Add default gateway to route"
        ip route add default via ${default_gateway} dev eth0
    elif [ "$1" == "up" ] ; then
        log "Remove default gateway from route"
        ip route del default via ${default_gateway} dev eth0
    fi

    log "Flush iptables rules"
    iptables -F

    if [ "$1" == "down" ] ; then
        log "Set route to DNS"
        iptables -A OUTPUT -o eth0 -p udp --dport 53 -j ACCEPT
        iptables -A OUTPUT -o eth0 -j REJECT
    else
        log "Setting subnet routes"
        for subnet in ${SUBNETS//,/ }; do
            # create a route to it and...
            ip route add ${subnet} via ${default_gateway} dev eth0
            # allow connections
            iptables -A INPUT -s ${subnet} -j ACCEPT
        done
    fi

    log "Set default routes"
    iptables -A INPUT -j ACCEPT -m state --state ESTABLISHED
    iptables -A INPUT -i lo -j ACCEPT
    iptables -A OUTPUT -o lo -j ACCEPT
}

function onVpnUp() {
    source ${VAR_DIR}/env

    until ip address show dev tun0 &> /dev/null ; do
        log "waiting..."
        sleep 1
    done

    # Start http proxy
    runTinyProxy&

    # Start socks proxy
    runSockd&

    # Configure routes
    configureRoutes up
}

function onVpnDown() {
    source ${VAR_DIR}/env

    log "Killing socks proxy"
    pkill -x sockd
    log "Killing http proxy"
    pkill -x tinyproxy

    while : ; do
        # Configure routes
        configureRoutes down

        server=$(grep '^remote ' ${VAR_DIR}/${VPN_CONF} | awk '{print $2}')
        nslookup $server &> /dev/null
        if [ $? -eq 0 ] ; then
            break
        fi
        log "Unable to reach server. Trying again..."
        sleep 1
    done
}

function saveEnv() {
    log "Saving env variables"

    env > ${VAR_DIR}/env

    default_gateway=$(ip r | grep 'default via' | cut -d " " -f 3)
    echo -e "default_gateway=${default_gateway}" >> ${VAR_DIR}/env
    local contents=$(cat ${VAR_DIR}/env)
    log "Save env:%n${contents}"
}

function configChanged() {
    diff ${CONFIG_DIR}/$1 ${VAR_DIR}/$1 &> /dev/null
    result=$?
    if [ $result -ne 0 ] ; then
        log "$1 has changed!"
    fi
    echo $result
}

# function internetAlive() {
#     ping -c 1 -A google.com
#     result=$?
#     if [ $result -ne 0 ] ; then
#         log "Internet is down!"
#     fi
#     echo $result
# }

function monitor() {
    log "Running monitor process..."

    task="monitor"
    while [ "$shutting_down" == "no" ] ; do
        if [ $(configChanged vpn.ovpn) -ne 0 -o $(configChanged vpn.auth) -ne 0 ] ; then
            saveVpnConfig
            log "Stopping openvpn"
        	pkill -x openvpn
        fi
        sleep ${RETRY_INTERVAL}
    done

    log "Exiting monitor"
}

function startVpn() {
    # Clear logs
    rm -f ${VAR_DIR}/*

    log "Starting OpenVPN-Proxy"
    mkdir -p ${VAR_DIR} ${CONFIG_DIR} &> /dev/null

    log "Configure sigterm handler"
    trap sigtermHandler SIGTERM

    if [ ! -f ${CONFIG_DIR}/${VPN_CONF} -o ! -f ${CONFIG_DIR}/${VPN_AUTH} ] ; then
        log "Invalid VPN config"
        exit 1
    fi

    # Save/load env variables
    if [ -f ${VAR_DIR}/env ] ; then
        source ${VAR_DIR}/env
    else
        saveEnv
    fi

    # Configure routes
    configureRoutes init

    # Save config files to var
    saveVpnConfig

    # Run monitor process
    monitor ${openvpn_pid} &
    monitor_pid=$!
    log "Monitor started with PID: ${monitor_pid}"

    while [ "$shutting_down" == "no" ] ; do
        log "Call openvpn"
        openvpn --config ${VAR_DIR}/${VPN_CONF} \
            --auth-nocache \
            --connect-retry-max 10 \
            --auth-user-pass ${VAR_DIR}/${VPN_AUTH} \
            --status ${VAR_DIR}/openvpn.status 15 \
            --log ${VAR_DIR}/openvpn.log \
            --pull-filter ignore "route-ipv6" \
            --pull-filter ignore "ifconfig-ipv6" \
            --script-security 2 \
            --up-delay --up ${VPN_UP_SCRIPT} \
            --down ${VPN_DOWN_SCRIPT} \
            --up-restart \
            --ping-restart ${RETRY_INTERVAL} \
            --group openvpn \
            --redirect-gateway autolocal \
            --cd ${VAR_DIR} &

        openvpn_pid=$!
        log "Openvpn started with PID: ${openvpn_pid}"

        wait $openvpn_pid

        if [ "$shutting_down" == "yes" ] ; then
            log "Shutdown encountered"
            break
        fi

        log "VPN down. Retrying openvpn connection after 5 seconds..."
        sleep ${RETRY_INTERVAL};
    done

    log "Exiting..."
}

function runTinyProxy() {
    log "Starting http proxy..."

    task="tinyproxy-wrapper"
    if [ "${HTTP_PROXY}" != "on" ] ; then
        log "HTTP_PROXY is ${HTTP_PROXY}"
        return
    fi

    until ip a show dev tun0 > /dev/null 2>&1 ; do
        sleep 1
    done

    if [ ! -f "${VAR_DIR}/${TINYPROXY_CONF}" ] ; then
        cp ${ETC_DIR}/${TINYPROXY_CONF} ${VAR_DIR}/${TINYPROXY_CONF}

        if [ ${PROXY_USERNAME} ] ; then
            if [ ${PROXY_PASSWORD} ] ; then
                log "Configuring proxy authentication"
                echo -e "\nBasicAuth ${PROXY_USERNAME} ${PROXY_PASSWORD}" >> ${VAR_DIR}/${TINYPROXY_CONF}
            else
                log "WARNING: Proxy username supplied without password. Starting HTTP proxy without credentials"
            fi
        fi
    fi

    # update IP
    local addr_eth=$(ip a show dev eth0 | grep "inet " | cut -d " " -f 6 | cut -d "/" -f 1)
    local addr_tun=$(ip a show dev tun0 | grep "inet " | cut -d " " -f 6 | cut -d "/" -f 1)
    sed -i \
        -e "/Listen.*/c Listen $addr_eth" \
        -e "/Bind.*/c Bind $addr_tun" \
        ${VAR_DIR}/${TINYPROXY_CONF}

    tinyproxy -d -c ${VAR_DIR}/${TINYPROXY_CONF} >> ${VAR_DIR}/${TINYPROXY_LOG} 2>&1 &
    tinyproxy_pid=$!
    log "Started Tinyproxy HTTP proxy server with PID: ${tinyproxy_pid}"

    wait $tinyproxy_pid

    log "Tinyproxy HTTP proxy server exited!"
}

function runSockd() {
    log "Starting socks proxy..."

    task="sockd-wrapper"
    if [ "$SOCKS_PROXY" != "on" ] ; then
        log "SOCKS_PROXY is $SOCKS_PROXY"
        return
    fi

    if [ ! -f "${VAR_DIR}/${SOCKD_CONF}" ] ; then
        cp ${ETC_DIR}/${SOCKD_CONF} ${VAR_DIR}/${SOCKD_CONF}

        if [ $PROXY_USERNAME ] ; then
            if [ $PROXY_PASSWORD ] ; then
                log "Configuring proxy authentication"
                adduser -S -D -g $PROXY_USERNAME -H -h /dev/null $PROXY_USERNAME
                echo "$PROXY_USERNAME:$PROXY_PASSWORD" | chpasswd 2> /dev/null
                sed -i 's/socksmethod: none/socksmethod: username/' ${VAR_DIR}/${SOCKD_CONF}
            else
                log "WARNING: Proxy username supplied without password. Starting SOCKS proxy without credentials"
            fi
        fi
    fi

    sockd -f ${VAR_DIR}/${SOCKD_CONF} >> ${VAR_DIR}/${SOCKD_LOG} 2>&1 &
    sockd_pid=$!
    log "Started Dante SOCKS proxy server with PID: ${sockd_pid}"

    wait ${sockd_pid}

    log "Dante SOCKS proxy server exited!"
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
