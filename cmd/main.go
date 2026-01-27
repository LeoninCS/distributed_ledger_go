package main

import (
	"fmt"
	"log"
	"path/filepath"

	"distributed_ledger_go/config"
	"distributed_ledger_go/internal/api"
	"distributed_ledger_go/internal/service"
	"distributed_ledger_go/internal/store"
	"distributed_ledger_go/internal/txVerify"

	"github.com/dgraph-io/badger/v3"
)

func main() {
	// 加载配置
	cfg, err := config.Load(filepath.Join("config", "config.yml"))
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	// 配置数据库
	opts := badger.DefaultOptions(cfg.DataDir)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalf("open badger: %v", err)
	}
	defer db.Close()

	s := store.NewStore(db)
	auditStore := store.NewStore(db)
	// 初始化服务
	accountSvc := service.NewAccountService(s)
	auditSvc := service.NewAuditService(auditStore)
	if err := auditSvc.VerifyChain(); err != nil {
		log.Fatalf("audit chain verification failed: %v", err)
	}
	validator := txVerify.NewValidator(s)
	txSvc := service.NewTransactionService(s, validator, auditSvc)

	server := api.NewServer(accountSvc, txSvc, auditSvc)
	addr := fmt.Sprintf(":%d", cfg.HTTPPort)
	log.Printf("HTTP server listening on %s", addr)
	if err := server.ListenAndServe(addr); err != nil {
		log.Fatalf("http server: %v", err)
	}
}
