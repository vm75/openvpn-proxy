#!/bin/sh

SCRIPT_DIR=$(dirname "${0}")
VAR_DIR=/data/var

function monitor() {
    while : ; do
		chown proxy:proxy ${VAR_DIR}/*
        sleep 3
    done
}

main() {
	cat /etc/passwd | grep -e '^proxy:' &> /dev/null

	if [ $? -ne 0 ] ; then
		addgroup -S -g ${PGID:-1000} proxy && \
		adduser -S -u ${PUID:-1000} -D -G proxy -g "proxy" -H -h /data -s /sbin/nologin proxy
	fi

	# Clear logs
	rm -f ${VAR_DIR}/*

	monitor&
    monitor_pid=$!

	su -s /bin/sh -c "${SCRIPT_DIR}/openvpn-proxy.sh" proxy

	kill -TERM ${monitor_pid}
}

main "$@"
