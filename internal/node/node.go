package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"distributed_ledger_go/config"
	"distributed_ledger_go/internal/api"
	"distributed_ledger_go/internal/service"
	"distributed_ledger_go/internal/store"
	"distributed_ledger_go/internal/txVerify"
	"distributed_ledger_go/internal/types"

	"github.com/dgraph-io/badger/v3"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

const (
	commandTransaction = "transaction"
)

// Node 表示一个运行中的账本节点（后续会逐步扩展 Raft 能力）。
type Node struct {
	cfg        *config.Config
	db         *badger.DB
	server     *api.Server
	accountSvc *service.AccountService
	txSvc      *service.TransactionService
	auditSvc   *service.AuditService

	raftNode *raft.Raft
}

type raftCommand struct {
	Type        string             `json:"type"`
	Transaction *types.Transaction `json:"transaction,omitempty"`
}

type fsm struct {
	txSvc *service.TransactionService
}

type fsmSnapshot struct{}

func NewNode(cfg *config.Config) (*Node, error) {
	opts := badger.DefaultOptions(cfg.DataDir)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	storeDB := store.NewStore(db)
	auditStore := store.NewStore(db)
	accountSvc := service.NewAccountService(storeDB)
	auditSvc := service.NewAuditService(auditStore)
	if err := auditSvc.VerifyChain(); err != nil {
		db.Close()
		return nil, err
	}
	validator := txVerify.NewValidator(storeDB)
	txSvc := service.NewTransactionService(storeDB, validator, auditSvc)

	n := &Node{
		cfg:        cfg,
		db:         db,
		accountSvc: accountSvc,
		txSvc:      txSvc,
		auditSvc:   auditSvc,
	}

	if err := n.initRaft(); err != nil {
		n.Close()
		return nil, err
	}

	n.server = api.NewServer(accountSvc, n.proposeTransaction, auditSvc)
	return n, nil
}

// 初始化Raft节点
func (n *Node) initRaft() error {
	if err := os.MkdirAll(n.cfg.RaftDir, 0o755); err != nil {
		return err
	}

	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(n.cfg.NodeID)

	fsm := &fsm{txSvc: n.txSvc}
	// 日志存储
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(n.cfg.RaftDir, "raft-log.bolt"))
	if err != nil {
		return err
	}
	// 稳定存储
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(n.cfg.RaftDir, "raft-stable.bolt"))
	if err != nil {
		return err
	}
	// 快照存储
	snapStore, err := raft.NewFileSnapshotStore(n.cfg.RaftDir, 1, os.Stdout)
	if err != nil {
		return err
	}
	// 网络传输
	transport, err := raft.NewTCPTransport(n.cfg.RaftBind, nil, 3, 10*time.Second, os.Stdout)
	if err != nil {
		return err
	}

	hasState, err := raft.HasExistingState(logStore, stableStore, snapStore)
	if err != nil {
		return err
	}

	raftNode, err := raft.NewRaft(config, fsm, logStore, stableStore, snapStore, transport)
	if err != nil {
		return err
	}

	if !hasState {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(n.cfg.NodeID),
					Address: transport.LocalAddr(),
				},
			},
		}
		future := raftNode.BootstrapCluster(configuration)
		if err := future.Error(); err != nil {
			return err
		}
	}

	n.raftNode = raftNode
	return nil
}

// 启动节点
func (n *Node) Start() error {
	addr := fmt.Sprintf(":%d", n.cfg.HTTPPort)
	log.Printf("node %s listening on %s (raft bind %s, data=%s, raft=%s)", n.cfg.NodeID, addr, n.cfg.RaftBind, n.cfg.DataDir, n.cfg.RaftDir)
	return n.server.ListenAndServe(addr)
}

// 关闭节点
func (n *Node) Close() error {
	if n.raftNode != nil {
		future := n.raftNode.Shutdown()
		_ = future.Error()
	}
	if n.db != nil {
		return n.db.Close()
	}
	return nil
}

// 把一个交易请求包装成 Raft 能够理解的命令，并请求集群达成一致。
func (n *Node) proposeTransaction(tx *types.Transaction) error {
	if n.raftNode == nil {
		return errors.New("raft not initialized")
	}
	// 包装交易（序列化）
	payload, err := json.Marshal(raftCommand{
		Type:        commandTransaction,
		Transaction: tx,
	})
	if err != nil {
		return err
	}
	// 提案
	future := n.raftNode.Apply(payload, 5*time.Second)
	if err := future.Error(); err != nil {
		return err
	}
	// 获取状态机响应
	if respErr, ok := future.Response().(error); ok && respErr != nil {
		return respErr
	}
	return nil
}

func (f *fsm) Apply(logEntry *raft.Log) interface{} {
	var cmd raftCommand
	if err := json.Unmarshal(logEntry.Data, &cmd); err != nil {
		return err
	}
	switch cmd.Type {
	case commandTransaction:
		if cmd.Transaction == nil {
			return errors.New("nil transaction")
		}
		return f.txSvc.Apply(*cmd.Transaction)
	default:
		return fmt.Errorf("unknown command: %s", cmd.Type)
	}
}

func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	return &fsmSnapshot{}, nil
}

func (f *fsm) Restore(rc io.ReadCloser) error {
	defer rc.Close()
	return nil
}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	if _, err := sink.Write([]byte{}); err != nil {
		_ = sink.Cancel()
		return err
	}
	return sink.Close()
}

func (s *fsmSnapshot) Release() {}
