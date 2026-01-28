package main

import (
	"log"
	"path/filepath"

	"distributed_ledger_go/config"
	"distributed_ledger_go/internal/node"
)

func main() {
	cfg, err := config.Load(filepath.Join("config", "config.yml"))
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	n, err := node.NewNode(cfg)
	if err != nil {
		log.Fatalf("init node: %v", err)
	}
	defer n.Close()
	if err := n.Start(); err != nil {
		log.Fatalf("node exit: %v", err)
	}
}
