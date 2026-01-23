package service

import (
	"distributed_ledger_go/internal/store"
	"distributed_ledger_go/internal/types"
)

// 封装账户层面的读写操作。
type AccountService struct {
	store *store.Store
}

func NewAccountService(s *store.Store) *AccountService {
	return &AccountService{store: s}
}

// 在 KV 中注册账户，并按需设置初始角色。
func (svc *AccountService) Register(address string, role string) error {
	if err := svc.store.RegisterAccount(address); err != nil {
		return err
	}
	if role != "" && role != types.RoleUser {
		return svc.store.SetRole(address, role)
	}
	return nil
}

// 调整账户角色权限。
func (svc *AccountService) GrantRole(address, role string) error {
	return svc.store.SetRole(address, role)
}

// 读取账户详情。
func (svc *AccountService) GetAccount(address string) (*types.Account, error) {
	return svc.store.GetAccount(address)
}
