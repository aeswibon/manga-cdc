package netguard

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

var allowedFlareSolverrHosts = map[string]struct{}{
	"localhost":    {},
	"127.0.0.1":    {},
	"::1":          {},
	"flaresolverr": {},
}

// ValidateFlareSolverrURL ensures FlareSolverr is only reachable on trusted internal endpoints.
func ValidateFlareSolverrURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid FLARESOLVERR_URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("FLARESOLVERR_URL must use http or https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("FLARESOLVERR_URL must include a host")
	}
	if parsed.User != nil {
		return fmt.Errorf("FLARESOLVERR_URL must not include credentials")
	}

	host := strings.ToLower(parsed.Hostname())
	if _, ok := allowedFlareSolverrHosts[host]; ok {
		return nil
	}

	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return nil
		}
		return fmt.Errorf("FLARESOLVERR_URL must not point to a public address")
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("FLARESOLVERR_URL host could not be resolved: %w", err)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			continue
		}
		return fmt.Errorf("FLARESOLVERR_URL must not resolve to a public address")
	}
	return nil
}
