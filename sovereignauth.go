package sovereignauth

import (
	"net/http"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(SovereignAuth{})
}

type SovereignAuth struct {
	APIKeys          []string `json:"api_keys,omitempty"`
	TailscaleAuth    bool     `json:"tailscale_auth,omitempty"`
	TrustedNetworks  []string `json:"trusted_networks,omitempty"`
	logger           *zap.Logger
}

func (SovereignAuth) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.sovereign_auth",
		New: func() caddy.Module { return new(SovereignAuth) },
	}
}

func (m *SovereignAuth) Provision(ctx caddy.Context) error {
	m.logger = ctx.Logger()
	return nil
}

func (m *SovereignAuth) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	clientIP := r.RemoteAddr
	for _, cidr := range m.TrustedNetworks {
		if strings.HasPrefix(clientIP, strings.Split(cidr, "/")[0]) {
			return next.ServeHTTP(w, r)
		}
	}

	if m.TailscaleAuth {
		if tailnet := r.Header.Get("X-Tailscale-User-Login"); tailnet != "" {
			m.logger.Info("tailscale auth", zap.String("user", tailnet))
			return next.ServeHTTP(w, r)
		}
	}

	if len(m.APIKeys) > 0 {
		authHeader := r.Header.Get("Authorization")
		apiKey := r.Header.Get("X-API-Key")
		token := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else if apiKey != "" {
			token = apiKey
		}
		for _, validKey := range m.APIKeys {
			if token == validKey {
				return next.ServeHTTP(w, r)
			}
		}
	}

	w.Header().Set("WWW-Authenticate", "Bearer realm=\"sovereign\"")
	http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	return nil
}
