package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	NodeID    string   `yaml:"node_id"`
	DataDir   string   `yaml:"data_dir"`
	HTTPPort  int      `yaml:"http_port"`
	RaftDir   string   `yaml:"raft_dir"`
	RaftBind  string   `yaml:"raft_bind"`
	RaftPeers []string `yaml:"raft_peers"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}

	if cfg.DataDir == "" {
		cfg.DataDir = "./data"
	}
	if cfg.RaftDir == "" {
		cfg.RaftDir = "./raft"
	}
	if cfg.RaftBind == "" {
		cfg.RaftBind = "127.0.0.1:7000"
	}
	if cfg.HTTPPort == 0 {
		cfg.HTTPPort = 8080
	}
	if cfg.NodeID == "" {
		return nil, errors.New("node_id is required")
	}

	return &cfg, nil
}
