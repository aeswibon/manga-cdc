package watchlist

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

var allowedWatchlistHosts = map[string]struct{}{
	"raw.githubusercontent.com": {},
	"github.com":                {},
	"gitlab.com":                {},
	"bitbucket.org":             {},
}

func ValidateRemoteURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid watchlist URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("watchlist URL must use https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("watchlist URL must include a host")
	}
	host := strings.ToLower(parsed.Hostname())
	if _, ok := allowedWatchlistHosts[host]; !ok {
		return fmt.Errorf("watchlist host %q is not allowlisted", host)
	}
	if err := rejectPrivateIP(host); err != nil {
		return err
	}
	return nil
}

func rejectPrivateIP(host string) error {
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("watchlist URL resolves to non-public address")
		}
		return nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("watchlist URL host could not be resolved: %w", err)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("watchlist URL resolves to non-public address")
		}
	}
	return nil
}
