#!/bin/sh

scriptName=$(basename $0)
scriptDir=$(dirname $(realpath $0))
shutting_down=no

rm /data/var/*.log

function cleanup {
    shutting_down=yes
    # When you run `docker stop` or any equivalent, a SIGTERM signal is sent to PID 1.
    # A process running as PID 1 inside a container is treated specially by Linux:
    # it ignores any signal with the default action. As a result, the process will
    # not terminate on SIGINT or SIGTERM unless it is coded to do so. Because of this,
    # I've defined behavior for when SIGINT and SIGTERM is received.
    if [ $healthcheck_child ]; then
        echo "Stopping healthcheck script..."
        kill -TERM $healthcheck_child
    fi

    if [ $openvpn_child ]; then
        echo "Stopping OpenVPN..."
        kill -TERM $openvpn_child
    fi

    sleep 1
    echo "Exiting."
    exit 0
}

function ensureVpnConfig() {
    config_file=$1
    if [ -f ${config_file} ] ; then
        dos2unix ${config_file}
    fi
}

mkdir -p /data/var /data/config

env > /data/var/env

config_file=/data/config/vpn.ovpn

ensureVpnConfig ${config_file}

if [ ! -f ${config_file} ] ; then
    echo -e "No VPN config file available.\n"
    exit 1
fi

echo -e "Starting OpenVPN client.\n"

default_gateway=$(ip r | grep 'default via' | cut -d " " -f 3)

echo "default_gateway=${default_gateway}" >> /data/var/env

iptables -F
iptables -A INPUT -j ACCEPT -m state --state ESTABLISHED
for subnet in ${SUBNETS//,/ }; do
    # create a route to it and...
    # ip route add ${subnet} via ${default_gateway} dev eth0
    # allow connections
    iptables -A INPUT -s ${subnet} -j ACCEPT
done

while [ "$shutting_down" == "no" ] ; do
    openvpn --config ${config_file} \
        --auth-nocache \
        --connect-retry-max 10 \
        --auth-user-pass /data/config/vpn.auth \
        --status /data/var/openvpn.status 15 \
        --log /data/var/openvpn.log \
        --pull-filter ignore "route-ipv6" \
        --pull-filter ignore "ifconfig-ipv6" \
        --up ${scriptDir}/vpn-up.sh \
        --down ${scriptDir}/vpn-down.sh \
        --up-restart \
        --group openvpn \
        --redirect-gateway autolocal \
        --cd /data/config &

    openvpn_child=$!

    wait $openvpn_child

    echo -e "VPN down. $shutting_down\n"

    if [ "$shutting_down" == "no" ] ; then
        echo -e "Retrying openvpn connection ...\n"
        sleep 15;
    fi
done

echo -e "Done!\n"
