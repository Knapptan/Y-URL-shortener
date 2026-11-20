package handler

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"

	"github.com/Knapptan/Y-URL-shortener/internal/service"
)

// statsResponse — тело ответа GET /api/internal/stats.
type statsResponse struct {
	URLs  int `json:"urls"`
	Users int `json:"users"`
}

// InternalStatsHandler — обработчик GET /api/internal/stats.
// Доступен только клиентам из доверенной подсети (trusted_subnet).
type InternalStatsHandler struct {
	service    *service.URLService
	trustedNet *net.IPNet // nil, если trusted_subnet не задан
}

// NewInternalStatsHandler создаёт обработчик.
// Если trustedSubnet == "" — доступ запрещён всем (403 на любой запрос).
// Если trustedSubnet некорректный CIDR — возвращается ошибка.
func NewInternalStatsHandler(svc *service.URLService, trustedSubnet string) (*InternalStatsHandler, error) {
	h := &InternalStatsHandler{service: svc}
	if trustedSubnet == "" {
		return h, nil // trustedNet остаётся nil → все запросы 403
	}
	_, ipnet, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		return nil, err
	}
	h.trustedNet = ipnet
	return h, nil
}

// ServeHTTP проверяет X-Real-IP и возвращает статистику.
func (h *InternalStatsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.trustedNet == nil {
		// trusted_subnet не задан → доступ запрещён всем.
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	ipStr := r.Header.Get("X-Real-IP")
	if ipStr == "" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if !h.trustedNet.Contains(ip) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	stats, err := h.service.Stats()
	if err != nil {
		slog.Error("Failed to get stats", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := statsResponse{URLs: stats.URLs, Users: stats.Users}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode stats response", "error", err)
	}
}
