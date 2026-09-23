package addr

import (
	"net"
	"strings"
)

var localSuffixes = []string{".local", ".localhost", ".test"}

func Local(hostport string) bool {
	host := hostport
	if split, _, err := net.SplitHostPort(hostport); err == nil {
		host = split
	}

	host = strings.ToLower(strings.Trim(host, "[]"))
	if host == "" {
		return false
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
	}
	if host == "localhost" {
		return true
	}

	for _, suffix := range localSuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}
