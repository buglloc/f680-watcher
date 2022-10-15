package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/buglloc/f680-watcher/internal/f860"
)

type Router struct {
	Upstream string `yaml:"upstream"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Config struct {
	Debug        bool                           `yaml:"debug"`
	Router       Router                         `yaml:"router"`
	NotifyScript string                         `yaml:"notify_script"`
	CheckPeriod  time.Duration                  `yaml:"check_period"`
	DHCPSources  map[string]f860.DHCPSourceKind `yaml:"dhcp_sources"`
}

func LoadConfig(cfgPath string) (*Config, error) {
	out := &Config{
		Debug: false,
		Router: Router{
			Upstream: "http://192.168.1.1",
			Username: "mgts",
			Password: os.Getenv("ROUTER_PASSWORD"),
		},
		CheckPeriod: 5 * time.Minute,
		DHCPSources: map[string]f860.DHCPSourceKind{
			"LAN1": f860.DHCPSourceKindInternet,
		},
	}

	if cfgPath == "" {
		return out, nil
	}

	f, err := os.Open(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open config file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := yaml.NewDecoder(f).Decode(&out); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return out, nil
}
