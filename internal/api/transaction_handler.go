package api

import (
	"net/http"

	"distributed_ledger_go/internal/txVerify"
	"distributed_ledger_go/internal/types"
	"distributed_ledger_go/pkg/crypto"

	"github.com/gin-gonic/gin"
)

type txRequest struct {
	Sender     string `json:"sender"`
	Receiver   string `json:"receiver"`
	Amount     uint64 `json:"amount"`
	Nonce      uint64 `json:"nonce"`
	PrivateKey string `json:"private_key"`
}

func (s *Server) handleMint(c *gin.Context) {
	s.handleTransaction(c, types.TxTypeMint)
}

func (s *Server) handleTransfer(c *gin.Context) {
	s.handleTransaction(c, types.TxTypeTransfer)
}

func (s *Server) handleTransaction(c *gin.Context, txType types.TxType) {
	var req txRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Sender == "" || req.Receiver == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sender & receiver required"})
		return
	}
	if req.PrivateKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "private_key required"})
		return
	}
	tx := types.Transaction{
		Type:     txType,
		Sender:   req.Sender,
		Receiver: req.Receiver,
		Amount:   req.Amount,
		Nonce:    req.Nonce,
	}

	priv, err := crypto.HexToPrivateKey(req.PrivateKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash := txVerify.TxHash(tx)
	sig, err := crypto.Sign(priv, hash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	tx.Signature = sig

	if err := s.txSvc.Apply(tx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
