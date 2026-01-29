package api

import (
	"sync"

	"distributed_ledger_go/internal/service"
	"distributed_ledger_go/internal/types"

	"github.com/gin-gonic/gin"
)

// Server 封装账户、交易、审计及 Raft 管理的 HTTP 接口。
type Server struct {
	engine     *gin.Engine
	accountSvc *service.AccountService
	auditSvc   *service.AuditService
	txSubmit   func(*types.Transaction) error
	joinFunc   func(string, string) (string, error)
	removeFunc func(string) (string, error)
	statusFunc func() map[string]interface{}
	mu         sync.Mutex
	hasCreator bool
}

func NewServer(account *service.AccountService, txSubmit func(*types.Transaction) error, audit *service.AuditService, joinFunc func(string, string) (string, error), removeFunc func(string) (string, error), statusFunc func() map[string]interface{}) *Server {
	engine := gin.Default()
	s := &Server{
		engine:     engine,
		accountSvc: account,
		auditSvc:   audit,
		txSubmit:   txSubmit,
		joinFunc:   joinFunc,
		removeFunc: removeFunc,
		statusFunc: statusFunc,
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.engine.Static("/ui", "./web")
	s.engine.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})
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
	s.engine.POST("/raft/join", s.handleRaftJoin)
	s.engine.POST("/raft/remove", s.handleRaftRemove)
	s.engine.GET("/raft/status", s.handleRaftStatus)
}

func (s *Server) ListenAndServe(addr string) error {
	return s.engine.Run(addr)
}
