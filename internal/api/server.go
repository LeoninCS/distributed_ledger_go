package api

import (
	"distributed_ledger_go/internal/service"
	"sync"

	"github.com/gin-gonic/gin"
)

// Server 使用 Gin 暴露账户、交易与审计接口。
type Server struct {
	engine     *gin.Engine
	accountSvc *service.AccountService
	txSvc      *service.TransactionService
	auditSvc   *service.AuditService
	mu         sync.Mutex
	hasCreator bool
}

func NewServer(account *service.AccountService, tx *service.TransactionService, audit *service.AuditService) *Server {
	engine := gin.Default()
	s := &Server{
		engine:     engine,
		accountSvc: account,
		txSvc:      tx,
		auditSvc:   audit,
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.engine.POST("/accounts/register", s.handleRegisterAccount)
	s.engine.GET("/accounts/:address", s.handleGetAccount)

	s.engine.POST("/transactions/mint", s.handleMint)
	s.engine.POST("/transactions/transfer", s.handleTransfer)

	s.engine.GET("/audit/:index", s.handleAuditEntry)
}

func (s *Server) ListenAndServe(addr string) error {
	return s.engine.Run(addr)
}
