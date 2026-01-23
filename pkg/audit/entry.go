package audit

import (
	"crypto/sha256"
	"distributed_ledger_go/internal/types"
	"encoding/binary"
	"errors"
)

// 序列化Entry
func EncodeEntry(e *types.Entry) ([]byte, error) {
	if e == nil {
		return nil, errors.New("nil entry")
	}
	if len(e.TxBytes) > int(^uint32(0)) {
		return nil, errors.New("tx too large")
	}

	out := make([]byte, 0, 8+32+32+4+len(e.TxBytes))

	var b8 [8]byte
	binary.LittleEndian.PutUint64(b8[:], e.Index)
	out = append(out, b8[:]...)

	out = append(out, e.PrevHash[:]...)
	out = append(out, e.EntryHash[:]...)

	var b4 [4]byte
	binary.LittleEndian.PutUint32(b4[:], uint32(len(e.TxBytes)))
	out = append(out, b4[:]...)

	out = append(out, e.TxBytes...)
	return out, nil
}

// 反序列化Entry
func DecodeEntry(b []byte) (*types.Entry, error) {
	const header = 8 + 32 + 32 + 4
	if len(b) < header {
		return nil, errors.New("invalid entry bytes: too short")
	}

	e := &types.Entry{}
	e.Index = binary.LittleEndian.Uint64(b[:8])
	copy(e.PrevHash[:], b[8:8+32])
	copy(e.EntryHash[:], b[8+32:8+32+32])

	txLen := binary.LittleEndian.Uint32(b[8+32+32 : 8+32+32+4])
	if len(b) != header+int(txLen) {
		return nil, errors.New("invalid entry bytes: length mismatch")
	}
	if txLen > 0 {
		e.TxBytes = make([]byte, txLen)
		copy(e.TxBytes, b[header:])
	}
	return e, nil
}

// 生成Entry的Hash
func AuditHash(index uint64, prev [32]byte, txBytes []byte) [32]byte {
	h := sha256.New()

	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], index)
	h.Write(buf[:])

	h.Write(prev[:])

	txHash := sha256.Sum256(txBytes)
	h.Write(txHash[:])

	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}
