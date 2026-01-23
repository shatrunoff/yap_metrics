package middleware

import (
	"net"
	"net/http"
)

// TrustedSubnetMiddleware проверяет, что IP из заголовка X-Real-IP входит в доверенную подсеть.
func TrustedSubnetMiddleware(trustedSubnet string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Если подсеть не задана, пропускаем все запросы
		if trustedSubnet == "" {
			return next
		}

		_, subnet, err := net.ParseCIDR(trustedSubnet)
		if err != nil {
			// Некорректный CIDR — пропускаем все запросы
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			realIP := r.Header.Get("X-Real-IP")
			if realIP == "" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			ip := net.ParseIP(realIP)
			if ip == nil || !subnet.Contains(ip) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
