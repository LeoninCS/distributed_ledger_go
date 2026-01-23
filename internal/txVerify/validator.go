package txVerify

import (
	"distributed_ledger_go/internal/store"
	"distributed_ledger_go/internal/types"
	"distributed_ledger_go/pkg/crypto"
	"encoding/hex"
	"errors"
	"fmt"
)

type Validator struct {
	store *store.Store
}

func NewValidator(s *store.Store) *Validator {
	return &Validator{store: s}
}

// 验证交易
func (v *Validator) ValidateTransaction(tx types.Transaction) error {
	if (tx.Type == types.TxTypeMint || tx.Type == types.TxTypeTransfer) && tx.Amount == 0 {
		return errors.New("invalid amount")
	}
	if err := v.VerifySignature(tx); err != nil {
		return fmt.Errorf("signature verification failed: %v", err)
	}

	switch tx.Type {
	case types.TxTypeMint:
		senderAcc, err := v.store.GetAccount(tx.Sender)
		if err != nil {
			return errors.New("sender account not found")
		}
		if tx.Nonce != senderAcc.Nonce+1 {
			return errors.New("invalid nonce: possible replay attack")
		}
		return v.validatePermission(tx.Type, tx.Sender)

	case types.TxTypeTransfer:
		senderAcc, err := v.store.GetAccount(tx.Sender)
		if err != nil {
			return errors.New("sender account not found")
		}
		if tx.Nonce != senderAcc.Nonce+1 {
			return errors.New("invalid nonce: possible replay attack")
		}
		return v.validateTransfer(senderAcc, tx)

	case types.TxTypeFreeze:
		return v.validatePermission(tx.Type, tx.Sender)

	case types.TxTypeUnfreeze:
		return v.validatePermission(tx.Type, tx.Sender)

	default:
		return errors.New("unknown transaction type")
	}
}

// 验证签名
func (v *Validator) VerifySignature(tx types.Transaction) error {
	pubBytes, err := hex.DecodeString(tx.Sender)
	if err != nil {
		return errors.New("invalid sender address: not hex")
	}

	hash := TxHash(tx)
	pubKey, err := crypto.BytesToPublishKey(pubBytes)
	if err != nil {
		return err
	}
	if !crypto.VerifyASN1Signature(pubKey, hash, tx.Signature) {
		return errors.New("ECDSA verification failed")
	}
	return nil
}

// 验证转账（账户是否冻结）
func (v *Validator) validateTransfer(sender *types.Account, tx types.Transaction) error {
	if sender.IsFrozen {
		return errors.New("account is frozen")
	}
	if sender.Balance < tx.Amount {
		return errors.New("insufficient balance")
	}
	return nil
}

// 验证是否是创始者或管理员
func (v *Validator) validatePermission(txType types.TxType, sender string) error {
	role, err := v.store.GetRole(sender)
	if err != nil {
		return err
	}
	if !types.CanRoleExecute(txType, role) {
		return fmt.Errorf("permission denied: %s cannot perform txType=%d", sender, txType)
	}
	return nil
}
