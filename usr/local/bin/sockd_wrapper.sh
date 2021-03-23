#!/bin/sh

source /data/var/env

scriptName=$(basename $0)
exec 1>> /data/var/${scriptName/.sh/.log}
exec 2>&1

if [ "$SOCKS_PROXY" != "on" ] ; then
    echo -e "SOCKS_PROXY is $SOCKS_PROXY\n"
    return
fi

until ip a | grep tun0 > /dev/null 2>&1 ; do
    sleep 1
done

config_file=/data/config/sockd.conf

if [ ! -f "${config_file}" ] ; then
    cp /usr/local/etc/sockd.conf ${config_file}

    if [ $PROXY_USERNAME ] ; then
        if [ $PROXY_PASSWORD ] ; then
            echo "Configuring proxy authentication."
            adduser -S -D -g $PROXY_USERNAME -H -h /dev/null $PROXY_USERNAME
            echo "$PROXY_USERNAME:$PROXY_PASSWORD" | chpasswd 2> /dev/null
            sed -i 's/socksmethod: none/socksmethod: username/' ${config_file}
        else
            echo "WARNING: Proxy username supplied without password. Starting SOCKS proxy without credentials."
        fi
    fi
fi

echo -e "Starting Dante SOCKS proxy server.\n"

sockd -f ${config_file}
