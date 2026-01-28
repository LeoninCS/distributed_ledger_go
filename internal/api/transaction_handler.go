package api

import (
	"encoding/json"
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

func (s *Server) handleFreeze(c *gin.Context) {
	s.handleTransaction(c, types.TxTypeFreeze)
}

func (s *Server) handleUnfreeze(c *gin.Context) {
	s.handleTransaction(c, types.TxTypeUnfreeze)
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
	if txType == types.TxTypeMint {
		receiverAcc, err := s.accountSvc.GetAccount(req.Receiver)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if receiverAcc.Role != types.RoleAdmin {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mint receiver must be ADMIN"})
			return
		}
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

	if err := s.txSubmit(&tx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type queryRequest struct {
	RequesterAddress string `json:"requester_address"`
	PrivateKey       string `json:"private_key"`
}

type transactionRecord struct {
	Index    uint64       `json:"index"`
	Type     types.TxType `json:"type"`
	Sender   string       `json:"sender"`
	Receiver string       `json:"receiver"`
	Amount   uint64       `json:"amount"`
	Nonce    uint64       `json:"nonce"`
}

func (s *Server) handleQueryTransactions(c *gin.Context) {
	var req queryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.RequesterAddress == "" || req.PrivateKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "requester_address and private_key required"})
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
	if addrHex != req.RequesterAddress {
		c.JSON(http.StatusForbidden, gin.H{"error": "private key mismatch"})
		return
	}
	acc, err := s.accountSvc.GetAccount(req.RequesterAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entries, err := s.auditSvc.ListEntries()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var result []transactionRecord
	totalMint := uint64(0)
	for _, e := range entries {
		var tx types.Transaction
		if err := json.Unmarshal(e.TxBytes, &tx); err != nil {
			continue
		}
		switch acc.Role {
		case types.RoleAdmin:
			if tx.Type == types.TxTypeTransfer {
				result = append(result, transactionRecord{e.Index, tx.Type, tx.Sender, tx.Receiver, tx.Amount, tx.Nonce})
			}
		case types.RoleCreator:
			if tx.Type == types.TxTypeMint {
				receiverAcc, err := s.accountSvc.GetAccount(tx.Receiver)
				if err == nil && receiverAcc.Role == types.RoleAdmin {
					result = append(result, transactionRecord{e.Index, tx.Type, tx.Sender, tx.Receiver, tx.Amount, tx.Nonce})
					totalMint += tx.Amount
				}
			}
		default:
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission"})
			return
		}
	}
	resp := gin.H{"transactions": result}
	if acc.Role == types.RoleCreator {
		resp["total_minted"] = totalMint
	}
	c.JSON(http.StatusOK, resp)
}
