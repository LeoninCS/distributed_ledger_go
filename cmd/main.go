package main

import (
	"flag"
	"log"

	"distributed_ledger_go/config"
	"distributed_ledger_go/internal/node"
)

func main() {
	configPath := flag.String("config", "config/config.yml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
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
