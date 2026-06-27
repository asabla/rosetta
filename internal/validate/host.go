package validate

import (
	"errors"
	"net"
	"strings"
)

func NormalizeHost(h string) (string, error) {
	h = strings.TrimSpace(strings.ToLower(strings.TrimSuffix(h, ".")))
	if h == "" {
		return "", errors.New("host is required")
	}
	if strings.Contains(h, "*") {
		return "", errors.New("wildcard hosts are not supported")
	}
	if ip := net.ParseIP(h); ip != nil {
		return ip.String(), nil
	}
	labels := strings.Split(h, ".")
	for _, l := range labels {
		if l == "" || len(l) > 63 {
			return "", errors.New("invalid hostname label")
		}
		for i, r := range l {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r == '-' && i > 0 && i < len(l)-1) {
				continue
			}
			return "", errors.New("invalid hostname character")
		}
	}
	return h, nil
}
