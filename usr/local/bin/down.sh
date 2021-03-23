#!/bin/sh

source /data/var/env

pkill socks
pkill tinyproxy
rm /data/var/openvpn-proxy.running

ip route add default via ${default_gateway} dev eth0
iptables -F
iptables -A OUTPUT -o eth0 -p udp --dport 53 -j ACCEPT
iptables -A INPUT -j ACCEPT -m state --state ESTABLISHED
iptables -A OUTPUT -o eth0 -j REJECT

iptables -A INPUT -i lo -j ACCEPT
iptables -A OUTPUT -o lo -j ACCEPT

for subnet in ${SUBNETS//,/ }; do
    # create a route to it and...
    ip route del ${subnet} via ${default_gateway} dev eth0
    # allow connections
    iptables -A INPUT -s ${subnet} -j ACCEPT
done
