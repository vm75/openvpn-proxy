#!/bin/sh

source /data/var/env

scriptName=$(basename $0)
exec 1>> /data/var/${scriptName/.sh/.log}
exec 2>&1

if [ "$HTTP_PROXY" != "on" ] ; then
    echo -e "HTTP_PROXY is $HTTP_PROXY\n"
    return
fi

until ip a | grep tun0 > /dev/null 2>&1 ; do
    sleep 1
done

config_file=/data/config/tinyproxy.conf

if [ ! -f "${config_file}" ] ; then
    cp /usr/local/etc/tinyproxy.conf ${config_file}

    if [ $PROXY_USERNAME ] ; then
        if [ $PROXY_PASSWORD ] ; then
            echo "Configuring proxy authentication."
            echo -e "\nBasicAuth $PROXY_USERNAME $PROXY_PASSWORD" >> ${config_file}
        else
            echo "WARNING: Proxy username supplied without password. Starting HTTP proxy without credentials."
        fi
    fi
fi

# update IP
addr_eth=$(ip a show dev eth0 | grep "inet " | cut -d " " -f 6 | cut -d "/" -f 1)
addr_tun=$(ip a show dev tun0 | grep "inet " | cut -d " " -f 6 | cut -d "/" -f 1)
sed -i \
    -e "/Listen/c Listen $addr_eth" \
    -e "/Bind/c Bind $addr_tun" \
    ${config_file}

echo -e "Starting Tinyproxy HTTP proxy server.\n"

tinyproxy -d -c ${config_file}
