#!/bin/sh

source /data/var/env

scriptDir=$(dirname $(realpath $0))
${scriptDir}/tinyproxy_wrapper.sh &
${scriptDir}/sockd_wrapper.sh &

# for subnet in ${SUBNETS//,/ }; do
#     # create a route to it and...
#     ip route add ${subnet} via ${default_gateway} dev eth0
#     # allow connections
#     iptables -A INPUT -s ${subnet} -j ACCEPT
# done

ip route del default via ${default_gateway} dev eth0
iptables -F
iptables -A INPUT -j ACCEPT -m state --state ESTABLISHED
iptables -A INPUT -i lo -j ACCEPT
iptables -A OUTPUT -o lo -j ACCEPT
# iptables -A INPUT -i tun0 -j ACCEPT
# iptables -A OUTPUT -o tun0 -j ACCEPT
# iptables -A INPUT -p tcp -m tcp --dport 8112 -j ACCEPT
# iptables -A INPUT -p tcp -m tcp --dport 6881 -j ACCEPT
# iptables -A INPUT -p udp -m udp --dport 6881 -j ACCEPT
# iptables -A INPUT -p tcp --match multiport --dport 49152:65535 -j ACCEPT
# iptables -A INPUT -p udp --match multiport --dport 49152:65535 -j ACCEPT

for subnet in ${SUBNETS//,/ }; do
    # create a route to it and...
    ip route add ${subnet} via ${default_gateway} dev eth0
    # allow connections
    iptables -A INPUT -s ${subnet} -j ACCEPT
done
# iptables -A OUTPUT -o eth0 -p udp --dport 53 -j ACCEPT
# iptables -A INPUT -j ACCEPT -m state --state ESTABLISHED
# iptables -A OUTPUT -o eth0 -j REJECT

touch /data/var/openvpn-proxy.running

# ping -c 10 -A google.com
# if [ $? -ne 0 ] ; then
# 	pkill openvpn
# fi