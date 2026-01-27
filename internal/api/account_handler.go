package api

import (
	"net/http"

	"distributed_ledger_go/internal/types"
	"distributed_ledger_go/pkg/crypto"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleRegisterAccount(c *gin.Context) {
	role := s.nextRole()
	priv, addr, err := crypto.GenerateKeyPair()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := s.accountSvc.Register(addr, role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	privHex, err := crypto.PrivateKeyToHex(priv)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"address":     addr,
		"private_key": privHex,
		"role":        role,
	})
}

func (s *Server) handleGetAccount(c *gin.Context) {
	addr := c.Param("address")
	if addr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address required"})
		return
	}
	acc, err := s.accountSvc.GetAccount(addr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, acc)
}

func (s *Server) nextRole() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasCreator {
		s.hasCreator = true
		return types.RoleCreator
	}
	return types.RoleUser
}
