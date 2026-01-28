package api

import (
	"distributed_ledger_go/internal/service"
	"distributed_ledger_go/internal/types"
	"sync"

	"github.com/gin-gonic/gin"
)

// Server 使用 Gin 暴露账户、交易与审计接口。
type Server struct {
	engine     *gin.Engine
	accountSvc *service.AccountService
	auditSvc   *service.AuditService
	txSubmit   func(*types.Transaction) error
	mu         sync.Mutex
	hasCreator bool
}

func NewServer(account *service.AccountService, txSubmit func(*types.Transaction) error, audit *service.AuditService) *Server {
	engine := gin.Default()
	s := &Server{
		engine:     engine,
		accountSvc: account,
		auditSvc:   audit,
		txSubmit:   txSubmit,
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.engine.POST("/accounts/register", s.handleRegisterAccount)
	s.engine.GET("/accounts/:address", s.handleGetAccount)
	s.engine.POST("/accounts/promote", s.handlePromoteAccount)
	s.engine.POST("/accounts/demote", s.handleDemoteAccount)

	s.engine.POST("/transactions/mint", s.handleMint)
	s.engine.POST("/transactions/transfer", s.handleTransfer)
	s.engine.POST("/transactions/freeze", s.handleFreeze)
	s.engine.POST("/transactions/unfreeze", s.handleUnfreeze)
	s.engine.POST("/transactions/query", s.handleQueryTransactions)

	s.engine.GET("/audit/:index", s.handleAuditEntry)
}

func (s *Server) ListenAndServe(addr string) error {
	return s.engine.Run(addr)
}
