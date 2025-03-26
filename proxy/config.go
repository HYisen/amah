package proxy

import (
	"bytes"
	"gopkg.in/yaml.v3"
	"net/url"
	"os"
)

type Config struct {
	ControlPlaneURL *url.URL
	FallbackURL     *url.URL
	AIAgentURL      *url.URL
}

func NewConfig(routerConfigFilepath string) (*Config, error) {
	data, err := os.ReadFile(routerConfigFilepath)
	if err != nil {
		return nil, err
	}

	rf := &RouterConfig{}
	if err := yaml.NewDecoder(bytes.NewReader(data)).Decode(rf); err != nil {
		return nil, err
	}

	serverNameToURL := make(map[string]string)
	for _, server := range rf.Servers {
		serverNameToURL[server.Name] = server.URL
	}

	var ret Config
	if ret.ControlPlaneURL, err = url.Parse(serverNameToURL["control-plane"]); err != nil {
		return nil, err
	}
	if ret.FallbackURL, err = url.Parse(serverNameToURL["fallback"]); err != nil {
		return nil, err
	}
	if ret.AIAgentURL, err = url.Parse(serverNameToURL["aiagent"]); err != nil {
		return nil, err
	}
	return &ret, nil
}

type Server struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type RouterConfig struct {
	Servers []Server `yaml:"servers"`
}
