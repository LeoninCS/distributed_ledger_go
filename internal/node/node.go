package node

import (
	"fmt"
	"log"

	"distributed_ledger_go/config"
	"distributed_ledger_go/internal/api"
	"distributed_ledger_go/internal/service"
	"distributed_ledger_go/internal/store"
	"distributed_ledger_go/internal/txVerify"

	"github.com/dgraph-io/badger/v3"
)

type Node struct {
	cfg        *config.Config
	db         *badger.DB
	server     *api.Server
	accountSvc *service.AccountService
	txSvc      *service.TransactionService
	auditSvc   *service.AuditService
}

func NewNode(cfg *config.Config) (*Node, error) {
	opts := badger.DefaultOptions(cfg.DataDir)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	storeDB := store.NewStore(db)
	auditStore := store.NewStore(db)
	accountSvc := service.NewAccountService(storeDB)
	auditSvc := service.NewAuditService(auditStore)
	if err := auditSvc.VerifyChain(); err != nil {
		db.Close()
		return nil, err
	}
	validator := txVerify.NewValidator(storeDB)
	txSvc := service.NewTransactionService(storeDB, validator, auditSvc)
	server := api.NewServer(accountSvc, txSvc, auditSvc)
	return &Node{
		cfg:        cfg,
		db:         db,
		server:     server,
		accountSvc: accountSvc,
		txSvc:      txSvc,
		auditSvc:   auditSvc,
	}, nil
}

func (n *Node) Start() error {
	addr := fmt.Sprintf(":%d", n.cfg.HTTPPort)
	log.Printf("node %s listening on %s (raft bind %s, data=%s, raft=%s)", n.cfg.NodeID, addr, n.cfg.RaftBind, n.cfg.DataDir, n.cfg.RaftDir)
	return n.server.ListenAndServe(addr)
}

func (n *Node) Close() error {
	if n.db != nil {
		return n.db.Close()
	}
	return nil
}
