package types

const (
	RoleCreator = "CREATOR"
	RoleAdmin   = "ADMIN"
	RoleUser    = "USER"
)

type Account struct {
	Address  string `json:"address"`
	Balance  uint64 `json:"balance"`
	Nonce    uint64 `json:"nonce"`
	IsFrozen bool   `json:"is_frozen"`
	Role     string `json:"role"`
}
