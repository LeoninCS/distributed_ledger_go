package service

import (
	"distributed_ledger_go/internal/store"
	"distributed_ledger_go/internal/txVerify"
	"distributed_ledger_go/internal/types"
)

// 负责交易的校验、审计以及状态落地。
type TransactionService struct {
	store     *store.Store
	validator *txVerify.Validator
	audit     *AuditService
}

func NewTransactionService(s *store.Store, v *txVerify.Validator, auditSvc *AuditService) *TransactionService {
	return &TransactionService{
		store:     s,
		validator: v,
		audit:     auditSvc,
	}
}

// 先通过 Validator 校验，再写审计，最后调用底层 Store。
func (svc *TransactionService) Apply(tx types.Transaction) error {
	if svc.validator != nil {
		if err := svc.validator.ValidateTransaction(tx); err != nil {
			return err
		}
	}

	if svc.audit != nil {
		if _, err := svc.audit.AppendTransaction(tx); err != nil {
			return err
		}
	}

	return svc.store.ApplyTransaction(tx)
}
