package api

import (
	"net/http"
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
	statusFunc func() map[string]interface{}
	mu         sync.Mutex
	hasCreator bool
}

type raftJoinRequest struct {
	NodeID      string `json:"node_id"`
	RaftAddress string `json:"raft_address"`
}

func NewServer(account *service.AccountService, txSubmit func(*types.Transaction) error, audit *service.AuditService, joinFunc func(string, string) (string, error), statusFunc func() map[string]interface{}) *Server {
	engine := gin.Default()
	s := &Server{
		engine:     engine,
		accountSvc: account,
		auditSvc:   audit,
		txSubmit:   txSubmit,
		joinFunc:   joinFunc,
		statusFunc: statusFunc,
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
	s.engine.POST("/raft/join", s.handleRaftJoin)
	s.engine.GET("/raft/status", s.handleRaftStatus)
}

func (s *Server) ListenAndServe(addr string) error {
	return s.engine.Run(addr)
}

func (s *Server) handleRaftJoin(c *gin.Context) {
	if s.joinFunc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "join unavailable"})
		return
	}
	var req raftJoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.NodeID == "" || req.RaftAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id and raft_address required"})
		return
	}
	leader, err := s.joinFunc(req.NodeID, req.RaftAddress)
	if err != nil {
		if leader != "" {
			c.Header("X-Raft-Leader", leader)
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleRaftStatus(c *gin.Context) {
	if s.statusFunc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "status unavailable"})
		return
	}
	c.JSON(http.StatusOK, s.statusFunc())
}
