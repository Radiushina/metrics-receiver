package middleware

import (
	"net"
	"net/http"
	"strings"
)

// TrustedSubnet проверяет X-Real-IP по CIDR.
// nil network — без ограничений.
func TrustedSubnet(network *net.IPNet) func(http.Handler) http.Handler {
	if network == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP")))
			if ip == nil || !network.Contains(ip) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
