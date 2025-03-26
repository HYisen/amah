package proxy

import (
	"net/http/httputil"
	"strings"
)

func New(cfg *Config) *httputil.ReverseProxy {
	aiPrefix := "/ai"
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetXForwarded()
			// I have searched it in Eta0, the v1 prefix algorithm shall work. Expand it if this becomes more complex.
			if strings.HasPrefix(r.In.URL.Path, "/v1") {
				r.SetURL(cfg.ControlPlaneURL)
			} else if strings.HasPrefix(r.In.URL.Path, aiPrefix) {
				r.Out.URL.Path = strings.TrimPrefix(r.Out.URL.Path, aiPrefix)
				r.Out.URL.RawPath = strings.TrimPrefix(r.Out.URL.RawPath, aiPrefix)
				r.SetURL(cfg.AIAgentURL)
			} else {
				r.SetURL(cfg.FallbackURL)
			}
		},
	}
}
