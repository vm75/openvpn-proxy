#!/bin/sh

SCRIPT_DIR=$(dirname "${0}")

${SCRIPT_DIR}/openvpn-proxy.sh --vpn-down &
