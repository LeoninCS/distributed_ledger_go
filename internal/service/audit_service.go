package service

import (
	"encoding/json"

	"distributed_ledger_go/internal/store"
	"distributed_ledger_go/internal/types"
)

// 封装审计链读写操作。
type AuditService struct {
	store *store.Store
}

func NewAuditService(s *store.Store) *AuditService {
	return &AuditService{store: s}
}

// 将交易序列化并写入审计链。
func (svc *AuditService) AppendTransaction(tx types.Transaction) (*types.Entry, error) {
	if svc.store == nil {
		return nil, nil
	}
	payload, err := json.Marshal(tx)
	if err != nil {
		return nil, err
	}
	return svc.store.Append(payload)
}

// 按索引读取审计条目。
func (svc *AuditService) GetEntry(index uint64) (*types.Entry, error) {
	if svc.store == nil {
		return nil, nil
	}
	return svc.store.GetEntry(index)
}

// 校验链式哈希完整性。
func (svc *AuditService) VerifyChain() error {
	if svc.store == nil {
		return nil
	}
	return svc.store.VerifyChain()
}

func (svc *AuditService) ListEntries() ([]*types.Entry, error) {
	if svc.store == nil {
		return nil, nil
	}
	return svc.store.ListEntries()
}
