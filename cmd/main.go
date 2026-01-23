package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"distributed_ledger_go/internal/store"
	"distributed_ledger_go/internal/txVerify"
	"distributed_ledger_go/internal/types"
	"distributed_ledger_go/pkg/crypto"

	"github.com/dgraph-io/badger/v3"
)

func main() {
	// 1) 打开 BadgerDB
	dataDir := filepath.Join(".", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("mkdir data dir: %v", err)
	}
	opts := badger.DefaultOptions(dataDir)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalf("open badger: %v", err)
	}
	defer db.Close()

	// 2) 构建 Store
	s := store.NewStore(db)

	// 2.1) 构建 Audit Store，并在启动时校验审计链
	a := store.NewStore(db)
	if err := a.VerifyChain(); err != nil {
		log.Fatalf("audit chain verification failed: %v", err)
	}

	// 3) 构建 Validator
	v := txVerify.NewValidator(s)

	// 4) 生成三个用户的密钥对
	creatorPrivKey, creatorAddr, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("generate creator key: %v", err)
	}
	fmt.Printf("CREATOR address: %s\n", creatorAddr)

	alicePrivKey, aliceAddr, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("generate alice key: %v", err)
	}
	fmt.Printf("alice address: %s\n", aliceAddr)

	_, allenAddr, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("generate allen key: %v", err)
	}
	fmt.Printf("allen address: %s\n", allenAddr)

	// 5) 注册三个账户
	mustRegister(s, creatorAddr)
	mustRegister(s, aliceAddr)
	mustRegister(s, allenAddr)

	// 6) 给 CREATOR 赋予角色权限
	mustGrantRole(s, creatorAddr, types.RoleCreator)

	// 7) CREATOR 铸币 5000 给 alice
	creatorAcc, err := s.GetAccount(creatorAddr)
	if err != nil {
		log.Fatalf("get creator: %v", err)
	}
	mintTx := types.Transaction{
		Type:     types.TxTypeMint,
		Sender:   creatorAddr,
		Receiver: aliceAddr,
		Amount:   5000,
		Nonce:    creatorAcc.Nonce + 1,
	}

	mintHash := txVerify.TxHash(mintTx)
	mintSig, err := crypto.Sign(creatorPrivKey, mintHash) // <<< 添加签名
	if err != nil {
		log.Fatalf("sign mint transaction: %v", err)
	}
	mintTx.Signature = mintSig

	if err := v.ValidateTransaction(mintTx); err != nil {
		log.Fatalf("validate mint failed: %v", err)
	}
	appendAuditOrDie(a, mintTx)
	if err := s.ApplyTransaction(mintTx); err != nil {
		log.Fatalf("apply mint failed: %v", err)
	}
	fmt.Println("\n=== After MINT (CREATOR -> alice 5000) ===")
	showAccount(s, creatorAddr, "CREATOR")
	showAccount(s, aliceAddr, "alice")
	showAccount(s, allenAddr, "allen")

	// 8) alice 转账 1000 给 allen
	aliceAcc, err := s.GetAccount(aliceAddr)
	if err != nil {
		log.Fatalf("get alice: %v", err)
	}
	transferTx := types.Transaction{
		Type:     types.TxTypeTransfer,
		Sender:   aliceAddr,
		Receiver: allenAddr,
		Amount:   1000,
		Nonce:    aliceAcc.Nonce + 1,
	}

	// 9) alice 对交易签名
	hash := txVerify.TxHash(transferTx)
	sig, err := crypto.Sign(alicePrivKey, hash)
	if err != nil {
		log.Fatalf("sign transaction: %v", err)
	}
	transferTx.Signature = sig

	// 10) 校验与应用
	if err := v.ValidateTransaction(transferTx); err != nil {
		log.Fatalf("validate transfer failed: %v", err)
	}
	appendAuditOrDie(a, transferTx)
	if err := s.ApplyTransaction(transferTx); err != nil {
		log.Fatalf("apply transfer failed: %v", err)
	}

	// 11) 打印最终结果
	fmt.Println("\n=== After TRANSFER (alice -> allen 1000) ===")
	showAccount(s, creatorAddr, "CREATOR")
	showAccount(s, aliceAddr, "alice")
	showAccount(s, allenAddr, "allen")
}

func mustRegister(s *store.Store, addr string) {
	if err := s.RegisterAccount(addr); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			log.Fatalf("register %s: %v", addr, err)
		}
	}
}

func mustGrantRole(s *store.Store, addr, role string) {
	if err := s.SetRole(addr, role); err != nil && !strings.Contains(err.Error(), "already") {
		log.Fatalf("set role %s for %s: %v", role, addr, err)
	}
}

func showAccount(s *store.Store, addr string, name string) {
	acc, err := s.GetAccount(addr)
	if err != nil {
		log.Fatalf("get %s: %v", name, err)
	}
	fmt.Printf("%-8s balance=%d nonce=%d frozen=%v\n", name, acc.Balance, acc.Nonce, acc.IsFrozen)
}

func appendAuditOrDie(a *store.Store, tx types.Transaction) {
	b, err := json.Marshal(tx)
	if err != nil {
		log.Fatalf("marshal tx for audit: %v", err)
	}
	if _, err := a.Append(b); err != nil {
		log.Fatalf("append audit log: %v", err)
	}
}
