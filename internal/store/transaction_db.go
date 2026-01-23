package store

import (
	"distributed_ledger_go/internal/types"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v3"
)

// 添加交易
func (s *Store) ApplyTransaction(tx types.Transaction) error {
	return s.db.Update(func(txn *badger.Txn) error {
		// 1. 发送者账户（必须已注册）
		senderAcc, err := s.getAccountWithTxn(txn, tx.Sender)
		if err != nil {
			return err
		}

		// 仅转账需要余额/冻结校验
		if tx.Type == types.TxTypeTransfer {
			if senderAcc.Balance < tx.Amount {
				return errors.New("insufficient balance")
			}
			if senderAcc.IsFrozen {
				return errors.New("sender account is frozen")
			}
			senderAcc.Balance -= tx.Amount
		}

		// 仅 MINT/TRANSFER 需要 nonce 校验与递增
		if tx.Type == types.TxTypeMint || tx.Type == types.TxTypeTransfer {
			if tx.Nonce != senderAcc.Nonce+1 {
				return fmt.Errorf("nonce mismatch: expected %d, got %d", senderAcc.Nonce+1, tx.Nonce)
			}
			senderAcc.Nonce++
		}

		// 2. 接收者账户（必须已注册）
		receiverAcc, err := s.getAccountWithTxn(txn, tx.Receiver)
		if err != nil {
			return err
		}

		// 3. 执行业务
		switch tx.Type {
		case types.TxTypeTransfer:
			receiverAcc.Balance += tx.Amount
		case types.TxTypeFreeze:
			receiverAcc.IsFrozen = true
		case types.TxTypeUnfreeze:
			receiverAcc.IsFrozen = false
		case types.TxTypeMint:
			receiverAcc.Balance += tx.Amount
		default:
			return errors.New("unknown transaction type")
		}

		// 4. 持久化
		if err := s.saveAccountWithTxn(txn, senderAcc); err != nil {
			return err
		}
		return s.saveAccountWithTxn(txn, receiverAcc)
	})
}

// 读取账户（未注册则报错）
func (s *Store) getAccountWithTxn(txn *badger.Txn, address string) (*types.Account, error) {
	key := []byte("acc:" + address)
	item, err := txn.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, fmt.Errorf("account not registered: %s", address)
		}
		return nil, err
	}
	var acc types.Account
	err = item.Value(func(val []byte) error {
		return json.Unmarshal(val, &acc)
	})
	return &acc, err
}

// 注册账户
func (s *Store) RegisterAccount(address string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key := []byte("acc:" + address)
		_, err := txn.Get(key)
		if err == nil {
			return errors.New("account already exists")
		}
		if err != badger.ErrKeyNotFound {
			return err
		}
		acc := &types.Account{
			Address:  address,
			Balance:  0,
			Nonce:    0,
			IsFrozen: false,
		}
		val, _ := json.Marshal(acc)
		return txn.Set(key, val)
	})
}

// 内部复用事务保存账户序列化数据
func (s *Store) saveAccountWithTxn(txn *badger.Txn, acc *types.Account) error {
	key := []byte("acc:" + acc.Address)
	val, _ := json.Marshal(acc)
	return txn.Set(key, val)
}
