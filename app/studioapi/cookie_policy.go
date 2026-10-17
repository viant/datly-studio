package studioapi

import (
	"net"
	"net/url"
)

// HTTP is supported only for local development; remote sessions always retain
// Secure cookies, even if a standalone caller supplies an insecure origin.
func secureSessionCookies(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme != "http" {
		return true
	}
	host := parsed.Hostname()
	return host != "localhost" && !net.ParseIP(host).IsLoopback()
}
