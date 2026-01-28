package store

import (
	"bytes"
	"distributed_ledger_go/internal/types"
	"distributed_ledger_go/pkg/audit"
	"encoding/binary"
	"errors"
	"fmt"

	badger "github.com/dgraph-io/badger/v3"
)

var (
	keyLastIndex = []byte("audit:lastIndex")
	keyLastHash  = []byte("audit:lastHash")
	keyEntryPref = []byte("audit:entry:")
)

// 将uint64 索引转成 8 字节小端
func entryKey(index uint64) []byte {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], index)
	return append(append([]byte{}, keyEntryPref...), b[:]...)
}

// 添加审计条目
func (s *Store) Append(txBytes []byte) (*types.Entry, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("nil audit store")
	}
	// 复制一份，避免调用方后续修改底层 slice 影响审计内容
	txCopy := append([]byte(nil), txBytes...)

	var appended *types.Entry
	err := s.db.Update(func(txn *badger.Txn) error {
		lastIndex, lastHash, err := loadLast(txn)
		if err != nil {
			return err
		}

		newIndex := lastIndex + 1
		e := &types.Entry{
			Index:    newIndex,
			PrevHash: lastHash,
			TxBytes:  txCopy,
		}
		e.EntryHash = audit.AuditHash(e.Index, e.PrevHash, e.TxBytes)

		enc, err := audit.EncodeEntry(e)
		if err != nil {
			return err
		}

		if err := txn.Set(entryKey(newIndex), enc); err != nil {
			return err
		}

		var b8 [8]byte
		binary.LittleEndian.PutUint64(b8[:], newIndex)
		if err := txn.Set(keyLastIndex, b8[:]); err != nil {
			return err
		}
		if err := txn.Set(keyLastHash, e.EntryHash[:]); err != nil {
			return err
		}

		appended = e
		return nil
	})
	return appended, err
}

// 获取审计条目
func (s *Store) GetEntry(index uint64) (*types.Entry, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("nil audit store")
	}
	var e *types.Entry
	err := s.db.View(func(txn *badger.Txn) error {
		it, err := txn.Get(entryKey(index))
		if err != nil {
			return err
		}
		return it.Value(func(val []byte) error {
			dec, derr := audit.DecodeEntry(val)
			if derr != nil {
				return derr
			}
			e = dec
			return nil
		})
	})
	return e, err
}

// 验证哈希链
func (s *Store) VerifyChain() error {
	if s == nil || s.db == nil {
		return errors.New("nil audit store")
	}
	return s.db.View(func(txn *badger.Txn) error {
		lastIndex, _, err := loadLast(txn)
		if err != nil {
			return err
		}
		if lastIndex == 0 {
			return nil
		}

		var prevHash [32]byte // genesis prevHash = 0
		for i := uint64(1); i <= lastIndex; i++ {
			item, err := txn.Get(entryKey(i))
			if err != nil {
				return fmt.Errorf("missing audit entry %d: %w", i, err)
			}
			var e *types.Entry
			err = item.Value(func(val []byte) error {
				dec, derr := audit.DecodeEntry(val)
				if derr != nil {
					return derr
				}
				e = dec
				return nil
			})
			if err != nil {
				return fmt.Errorf("decode audit entry %d: %w", i, err)
			}

			if e.Index != i {
				return fmt.Errorf("audit entry index mismatch: want %d got %d", i, e.Index)
			}
			if e.PrevHash != prevHash {
				return fmt.Errorf("audit chain broken at %d: prevHash mismatch", i)
			}

			want := audit.AuditHash(e.Index, e.PrevHash, e.TxBytes)
			if !bytes.Equal(want[:], e.EntryHash[:]) {
				return fmt.Errorf("audit chain broken at %d: entryHash mismatch", i)
			}

			prevHash = e.EntryHash
		}
		return nil
	})
}

// 读取lastIndex和lastHash
func loadLast(txn *badger.Txn) (uint64, [32]byte, error) {
	var lastIndex uint64
	var lastHash [32]byte

	// lastIndex
	if item, err := txn.Get(keyLastIndex); err == nil {
		if err := item.Value(func(v []byte) error {
			if len(v) != 8 {
				return errors.New("invalid lastIndex length")
			}
			lastIndex = binary.LittleEndian.Uint64(v)
			return nil
		}); err != nil {
			return 0, [32]byte{}, err
		}
	} else if err != badger.ErrKeyNotFound {
		return 0, [32]byte{}, err
	}

	// lastHash
	if item, err := txn.Get(keyLastHash); err == nil {
		if err := item.Value(func(v []byte) error {
			if len(v) != 32 {
				return errors.New("invalid lastHash length")
			}
			copy(lastHash[:], v)
			return nil
		}); err != nil {
			return 0, [32]byte{}, err
		}
	} else if err != badger.ErrKeyNotFound {
		return 0, [32]byte{}, err
	}

	return lastIndex, lastHash, nil
}

func (s *Store) ListEntries() ([]*types.Entry, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("nil audit store")
	}
	var entries []*types.Entry
	err := s.db.View(func(txn *badger.Txn) error {
		lastIndex, _, err := loadLast(txn)
		if err != nil {
			return err
		}
		for i := uint64(1); i <= lastIndex; i++ {
			item, err := txn.Get(entryKey(i))
			if err != nil {
				return err
			}
			var e *types.Entry
			err = item.Value(func(val []byte) error {
				dec, derr := audit.DecodeEntry(val)
				if derr != nil {
					return derr
				}
				e = dec
				return nil
			})
			if err != nil {
				return err
			}
			entries = append(entries, e)
		}
		return nil
	})
	return entries, err
}
