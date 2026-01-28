package api

import (
	"net/http"

	"distributed_ledger_go/internal/types"
	"distributed_ledger_go/pkg/crypto"

	"github.com/gin-gonic/gin"
)

type promoteRequest struct {
	CreatorAddress string `json:"creator_address"`
	TargetAddress  string `json:"target_address"`
	PrivateKey     string `json:"private_key"`
}

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

func (s *Server) handlePromoteAccount(c *gin.Context) {
	var req promoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.CreatorAddress == "" || req.TargetAddress == "" || req.PrivateKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "creator_address, target_address and private_key required"})
		return
	}
	priv, err := crypto.HexToPrivateKey(req.PrivateKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	addrHex, err := crypto.PublicKeyToHex(&priv.PublicKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if addrHex != req.CreatorAddress {
		c.JSON(http.StatusForbidden, gin.H{"error": "private key does not match creator address"})
		return
	}
	creator, err := s.accountSvc.GetAccount(req.CreatorAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if creator.Role != types.RoleCreator {
		c.JSON(http.StatusForbidden, gin.H{"error": "only creator can promote"})
		return
	}
	if _, err := s.accountSvc.GetAccount(req.TargetAddress); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.accountSvc.GrantRole(req.TargetAddress, types.RoleAdmin); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"target": req.TargetAddress, "role": types.RoleAdmin})
}

func (s *Server) handleDemoteAccount(c *gin.Context) {
	var req promoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.CreatorAddress == "" || req.TargetAddress == "" || req.PrivateKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "creator_address, target_address and private_key required"})
		return
	}
	priv, err := crypto.HexToPrivateKey(req.PrivateKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	addrHex, err := crypto.PublicKeyToHex(&priv.PublicKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if addrHex != req.CreatorAddress {
		c.JSON(http.StatusForbidden, gin.H{"error": "private key does not match creator address"})
		return
	}
	creator, err := s.accountSvc.GetAccount(req.CreatorAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if creator.Role != types.RoleCreator {
		c.JSON(http.StatusForbidden, gin.H{"error": "only creator can demote"})
		return
	}
	if _, err := s.accountSvc.GetAccount(req.TargetAddress); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.accountSvc.GrantRole(req.TargetAddress, types.RoleUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"target": req.TargetAddress, "role": types.RoleUser})
}
