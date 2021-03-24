#!/bin/sh

source /data/var/env

scriptDir=$(dirname $(realpath $0))
${scriptDir}/tinyproxy_wrapper.sh &
${scriptDir}/sockd_wrapper.sh &

ip route del default via ${default_gateway} dev eth0
iptables -F
iptables -A INPUT -j ACCEPT -m state --state ESTABLISHED
iptables -A INPUT -i lo -j ACCEPT
iptables -A OUTPUT -o lo -j ACCEPT

for subnet in ${SUBNETS//,/ }; do
    # create a route to it and...
    ip route add ${subnet} via ${default_gateway} dev eth0
    # allow connections
    iptables -A INPUT -s ${subnet} -j ACCEPT
done

touch /data/var/openvpn-proxy.running
