package proxy

import (
	"log"
	"net/http/httputil"
	"net/url"
	"strings"
)

func New(basic *url.URL, other *url.URL) *httputil.ReverseProxy {
	aiagent, err := url.Parse("http://localhost:8640")
	if err != nil {
		log.Fatal(err)
	}
	aiPrefix := "/ai"
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetXForwarded()
			// I have searched it in Eta0, the v1 prefix algorithm shall work. Expand it if this becomes more complex.
			if strings.HasPrefix(r.In.URL.Path, "/v1") {
				r.SetURL(basic)
			} else if strings.HasPrefix(r.In.URL.Path, aiPrefix) {
				r.Out.URL.Path = strings.TrimPrefix(r.Out.URL.Path, aiPrefix)
				r.Out.URL.RawPath = strings.TrimPrefix(r.Out.URL.RawPath, aiPrefix)
				r.SetURL(aiagent)
			} else {
				r.SetURL(other)
			}
		},
	}
}
