package types

// 表示交易类型。
type TxType int

const (
	TxTypeMint TxType = iota
	TxTypeTransfer
	TxTypeFreeze
	TxTypeUnfreeze
)

var txPermissions = map[TxType][]string{
	TxTypeMint:     {RoleCreator},
	TxTypeTransfer: {RoleCreator, RoleAdmin, RoleUser},
	TxTypeFreeze:   {RoleCreator, RoleAdmin},
	TxTypeUnfreeze: {RoleCreator, RoleAdmin},
}

// 判断指定角色是否允许执行交易类型。
func CanRoleExecute(txType TxType, role string) bool {
	for _, allowed := range txPermissions[txType] {
		if allowed == role {
			return true
		}
	}
	return false
}

// 交易结构
type Transaction struct {
	Type      TxType
	Sender    string
	Receiver  string
	Amount    uint64
	Nonce     uint64
	Signature []byte
}
