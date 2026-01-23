package store

import (
	"distributed_ledger_go/internal/types"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v3"
)

const AccPrefix = "acc:"

// 更新账户余额
func (s *Store) UpdateAccount(address string, amount uint64) error {
	if s == nil || s.db == nil {
		return errors.New("nil store")
	}
	return s.db.Update(func(txn *badger.Txn) error {
		acc, err := s.getAccountWithTxn(txn, address)
		if err != nil {
			return err
		}
		acc.Balance = amount
		return s.saveAccountWithTxn(txn, acc)
	})
}

// 获取账户信息
func (s *Store) GetAccount(address string) (*types.Account, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("nil store")
	}

	var acc types.Account

	err := s.db.View(func(txn *badger.Txn) error {
		key := []byte(AccPrefix + address)

		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		// 提取并反序列化 Value
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &acc)
		})
	})

	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, fmt.Errorf("account not found: %s", address)
	}
	if err != nil {
		return nil, err
	}

	return &acc, nil
}

// 获取角色
func (s *Store) GetRole(address string) (string, error) {
	acc, err := s.GetAccount(address)
	if err != nil {
		return "", err
	}

	// 如果是新账户，默认角色是 USER
	if acc.Role == "" {
		return types.RoleUser, nil
	}

	return acc.Role, nil
}

// 设置角色
func (s *Store) SetRole(address, role string) error {
	if s == nil || s.db == nil {
		return errors.New("nil store")
	}
	return s.db.Update(func(txn *badger.Txn) error {
		acc, err := s.getAccountWithTxn(txn, address)
		if err != nil {
			return err
		}
		acc.Role = role
		return s.saveAccountWithTxn(txn, acc)
	})
}
