package txVerify

import (
	"bytes"
	"crypto/sha256"
	"distributed_ledger_go/internal/types"
	"encoding/binary"
)

// 生成交易哈希（不包含签名字段，避免循环依赖）
func TxHash(tx types.Transaction) []byte {
	res := new(bytes.Buffer)
	_ = binary.Write(res, binary.BigEndian, int32(tx.Type))
	_ = binary.Write(res, binary.BigEndian, []byte(tx.Sender))
	_ = binary.Write(res, binary.BigEndian, []byte(tx.Receiver))
	_ = binary.Write(res, binary.BigEndian, tx.Amount)
	_ = binary.Write(res, binary.BigEndian, tx.Nonce)

	hash := sha256.Sum256(res.Bytes())
	return hash[:]
}
