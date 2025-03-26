package proxy

import (
	"amah/client/auth"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
)

func New(cfg *Config, authClient *auth.Client) *httputil.ReverseProxy {
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

				if !authenticate(r.Out.URL.Path, r.In.Header.Get("token"), authClient) {
					r.SetURL(cfg.ControlPlaneURL)
					r.Out.Method = http.MethodGet
					r.Out.URL.Path = "/v0/forbidden"
				}
			} else {
				r.SetURL(cfg.FallbackURL)
			}
		},
	}
}

func authenticate(path string, token string, authClient *auth.Client) bool {
	rest, ok := strings.CutPrefix(path, "/v2/users/")
	if !ok {
		return false
	}
	head, _, _ := strings.Cut(rest, "/")
	uid, err := strconv.Atoi(head)
	if err != nil {
		return false
	}
	t, ok := authClient.FindValidToken(token)
	if !ok {
		return false
	}
	userID, ok := authClient.FindUserIDByUsername(t.Username)
	if !ok {
		return false
	}
	return userID == uid
}
