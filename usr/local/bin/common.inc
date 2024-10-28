#!/bin/sh

# exec 1>> /data/var/openvpn-proxy.log 2>&1

# script vars
SCRIPT_DIR=$(dirname "$0")
SCRIPT_NAME=$(basename "$0")

# dirs
ETC_DIR=/usr/local/etc
DATA_DIR=/data
CONFIG_DIR=${DATA_DIR}/config
VAR_DIR=${DATA_DIR}/var

# files
OPENVPNPROXY_LOG=${VAR_DIR}/openvpn-proxy.log
SAVED_VARS=${VAR_DIR}/env
VPN_CONF=${VAR_DIR}/vpn.ovpn
VPN_AUTH=${VAR_DIR}/vpn.auth

# when OPENVPNPROXY_LOG exceeds 1MB, rotate it
# maintain 5 rotated logs
function logRotate() {
    if [ ! -f ${OPENVPNPROXY_LOG} ] ; then
        return
    fi

    local size=$(du -k ${OPENVPNPROXY_LOG} | cut -f 1)
    if [ ${size} -gt 1024 ] ; then
        local i=5
        while [ ${i} -gt 0 ] ; do
            local j=$((i-1))
            if [ -f ${OPENVPNPROXY_LOG}.${i} ] ; then
                mv -f ${OPENVPNPROXY_LOG}.${j} ${OPENVPNPROXY_LOG}.${i}
            fi
            i=${j}
        done
        mv -f ${OPENVPNPROXY_LOG} ${OPENVPNPROXY_LOG}.0
        touch ${OPENVPNPROXY_LOG}
    fi
}

function log() {
    date +"%D %R ${task:-${SCRIPT_NAME}}: $@" >> ${OPENVPNPROXY_LOG}
    logRotate
}

function loadSavedVars() {
    if [ -f ${SAVED_VARS} ] ; then
        . ${SAVED_VARS}
    fi
}