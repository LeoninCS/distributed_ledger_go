package types

// 审计记录
type Entry struct {
	Index     uint64
	PrevHash  [32]byte
	TxBytes   []byte
	EntryHash [32]byte
}
