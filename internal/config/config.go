package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     Server     `yaml:"server"`
	ZLMediaKit ZLMediaKit `yaml:"zlmediakit"`
	Admin      Admin      `yaml:"admin"`
	Database   Database   `yaml:"database"`
}

type Server struct {
	Addr string `yaml:"addr"`
}

type ZLMediaKit struct {
	APIBase         string `yaml:"api_base"`
	Secret          string `yaml:"secret"`
	RTMPPushBase    string `yaml:"rtmp_push_base"`
	HTTPFLVPlayBase string `yaml:"http_flv_play_base"`
	App             string `yaml:"app"`
}

type Admin struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Database struct {
	Path string `yaml:"path"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}
	if cfg.ZLMediaKit.App == "" {
		cfg.ZLMediaKit.App = "live"
	}
	cfg.ZLMediaKit.APIBase = strings.TrimRight(cfg.ZLMediaKit.APIBase, "/")
	cfg.ZLMediaKit.RTMPPushBase = strings.TrimRight(cfg.ZLMediaKit.RTMPPushBase, "/")
	cfg.ZLMediaKit.HTTPFLVPlayBase = strings.TrimRight(cfg.ZLMediaKit.HTTPFLVPlayBase, "/")
	cfg.ZLMediaKit.App = strings.Trim(cfg.ZLMediaKit.App, "/")
	if cfg.Database.Path == "" {
		cfg.Database.Path = "./data.db1"
	}
	if cfg.ZLMediaKit.Secret == "" {
		return nil, fmt.Errorf("zlmediakit.secret is required")
	}
	if cfg.Admin.Username == "" || cfg.Admin.Password == "" {
		return nil, fmt.Errorf("admin.username and admin.password are required")
	}
	return &cfg, nil
}
