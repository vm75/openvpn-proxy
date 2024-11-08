package socks_proxy

import (
	"openvpn-proxy/core"
	"openvpn-proxy/utils"
	"os"
	"strings"
)

func updateConfig() error {
	content, err := os.ReadFile("/usr/local/etc/sockd.conf")

	if err != nil {
		return err
	}

	contentStr := string(content)

	if core.GlobalConfig.ProxyUsername != "" && core.GlobalConfig.ProxyPassword != "" {
		utils.CreateUser(core.GlobalConfig.ProxyUsername)
		contentStr = strings.Replace(contentStr, "socksmethod: none", "socksmethod: username", 1)
	}

	err = os.WriteFile(configFile, []byte(contentStr), 0644)

	if err != nil {
		return err
	}

	return nil
}
