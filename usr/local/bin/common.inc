#!/bin/sh

# exec 1>> /data/var/openvpn-proxy.log 2>&1

# script vars
SCRIPT_DIR=$(dirname "$0")
SCRIPT_NAME=$(basename "$0")

# dirs
ETC_DIR=/usr/local/etc
CONFIG_DIR=/data/config
VAR_DIR=/data/var

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
        log "Rotating log file"
        local i=5
        while [ ${i} -gt 0 ] ; do
            local j=$((i-1))
            if [ -f ${OPENVPNPROXY_LOG}.${j} ] ; then
                mv ${OPENVPNPROXY_LOG}.${j} ${OPENVPNPROXY_LOG}.${i}
            fi
            i=${j}
        done
        mv ${OPENVPNPROXY_LOG} ${OPENVPNPROXY_LOG}.0
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