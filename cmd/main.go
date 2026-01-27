package main

import (
	"fmt"
	"log"
	"path/filepath"

	"distributed_ledger_go/config"
	"distributed_ledger_go/internal/service"
	"distributed_ledger_go/internal/store"
	"distributed_ledger_go/internal/txVerify"
	"distributed_ledger_go/internal/types"
	"distributed_ledger_go/pkg/crypto"

	"github.com/dgraph-io/badger/v3"
)

func main() {
	// 加载配置
	cfg, err := config.Load(filepath.Join("config", "config.yml"))
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	// 配置数据库
	opts := badger.DefaultOptions(cfg.DataDir)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalf("open badger: %v", err)
	}
	defer db.Close()

	s := store.NewStore(db)
	auditStore := store.NewStore(db)
	// 初始化服务
	accountSvc := service.NewAccountService(s)
	auditSvc := service.NewAuditService(auditStore)
	if err := auditSvc.VerifyChain(); err != nil {
		log.Fatalf("audit chain verification failed: %v", err)
	}
	validator := txVerify.NewValidator(s)
	txSvc := service.NewTransactionService(s, validator, auditSvc)

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

	mustRegister(accountSvc, creatorAddr, types.RoleCreator)
	mustRegister(accountSvc, aliceAddr, "")
	mustRegister(accountSvc, allenAddr, "")

	creatorAcc, err := accountSvc.GetAccount(creatorAddr)
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
	mintSig, err := crypto.Sign(creatorPrivKey, mintHash)
	if err != nil {
		log.Fatalf("sign mint transaction: %v", err)
	}
	mintTx.Signature = mintSig

	if err := txSvc.Apply(mintTx); err != nil {
		log.Fatalf("apply mint failed: %v", err)
	}
	fmt.Println("\n=== After MINT (CREATOR -> alice 5000) ===")
	showAccount(accountSvc, creatorAddr, "CREATOR")
	showAccount(accountSvc, aliceAddr, "alice")
	showAccount(accountSvc, allenAddr, "allen")

	aliceAcc, err := accountSvc.GetAccount(aliceAddr)
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

	hash := txVerify.TxHash(transferTx)
	sig, err := crypto.Sign(alicePrivKey, hash)
	if err != nil {
		log.Fatalf("sign transaction: %v", err)
	}
	transferTx.Signature = sig

	if err := txSvc.Apply(transferTx); err != nil {
		log.Fatalf("apply transfer failed: %v", err)
	}

	fmt.Println("\n=== After TRANSFER (alice -> allen 1000) ===")
	showAccount(accountSvc, creatorAddr, "CREATOR")
	showAccount(accountSvc, aliceAddr, "alice")
	showAccount(accountSvc, allenAddr, "allen")
}

func mustRegister(svc *service.AccountService, addr, role string) {
	if err := svc.Register(addr, role); err != nil {
		log.Fatalf("register %s: %v", addr, err)
	}
}

func showAccount(svc *service.AccountService, addr string, name string) {
	acc, err := svc.GetAccount(addr)
	if err != nil {
		log.Fatalf("get %s: %v", name, err)
	}
	fmt.Printf("%-8s balance=%d nonce=%d frozen=%v\n", name, acc.Balance, acc.Nonce, acc.IsFrozen)
}
