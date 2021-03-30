#!/bin/sh

source /data/var/env

pkill socks
pkill tinyproxy
rm -f /data/var/openvpn-proxy.running

while : ; do
    route-reset.sh
    server=$(grep '^remote ' /data/config/vpn.ovpn  | awk '{print $2}')
    nslookup $server &> /dev/null
    if [ $? -eq 0 ] ; then
        break
    fi
    sleep 5
done

