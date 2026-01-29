package node

import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "time"

    "distributed_ledger_go/config"
    "distributed_ledger_go/internal/api"
    "distributed_ledger_go/internal/service"
    "distributed_ledger_go/internal/store"
    "distributed_ledger_go/internal/types"
    "distributed_ledger_go/internal/txVerify"

    "github.com/dgraph-io/badger/v3"
    "github.com/hashicorp/raft"
    raftboltdb "github.com/hashicorp/raft-boltdb"
)

const commandTransaction = "transaction"

// joinRequest 表示节点加入集群时提交的信息。
type joinRequest struct {
	NodeID      string `json:"node_id"`
	RaftAddress string `json:"raft_address"`
}

// Node 表示一个账本节点，封装业务服务与 Raft 复制。
type Node struct {
	cfg        *config.Config
	db         *badger.DB
	server     *api.Server
	accountSvc *service.AccountService
    txSvc      *service.TransactionService
    auditSvc   *service.AuditService

    raftNode *raft.Raft
    hasState bool
}

// NewNode 根据配置初始化业务服务与 Raft 实例。
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

    hasState, err := n.initRaft()
    if err != nil {
        n.Close()
        return nil, err
    }
    n.hasState = hasState
    if !hasState && !cfg.RaftBootstrap {
        if err := n.joinCluster(); err != nil {
            n.Close()
            return nil, err
        }
    }

    n.server = api.NewServer(accountSvc, n.proposeTransaction, auditSvc, n.handleJoinRequest, n.raftStatus)
    return n, nil
}

// initRaft 创建并配置 Raft 组件，返回是否存在旧状态。
func (n *Node) initRaft() (bool, error) {
    if err := os.MkdirAll(n.cfg.RaftDir, 0o755); err != nil {
        return false, err
    }

    rConfig := raft.DefaultConfig()
    rConfig.LocalID = raft.ServerID(n.cfg.NodeID)

    fsm := &fsm{txSvc: n.txSvc, db: n.db}

    logStore, err := raftboltdb.NewBoltStore(filepath.Join(n.cfg.RaftDir, "raft-log.bolt"))
    if err != nil {
        return false, err
    }
    stableStore, err := raftboltdb.NewBoltStore(filepath.Join(n.cfg.RaftDir, "raft-stable.bolt"))
    if err != nil {
        return false, err
    }
    snapStore, err := raft.NewFileSnapshotStore(n.cfg.RaftDir, 1, os.Stdout)
    if err != nil {
        return false, err
    }

    transport, err := raft.NewTCPTransport(n.cfg.RaftBind, nil, 3, 10*time.Second, os.Stdout)
    if err != nil {
        return false, err
    }

    hasState, err := raft.HasExistingState(logStore, stableStore, snapStore)
    if err != nil {
        return false, err
    }

    raftNode, err := raft.NewRaft(rConfig, fsm, logStore, stableStore, snapStore, transport)
    if err != nil {
        return false, err
    }

    if n.cfg.RaftBootstrap && !hasState {
        conf := raft.Configuration{
            Servers: []raft.Server{
                {
                    ID:      raft.ServerID(n.cfg.NodeID),
                    Address: transport.LocalAddr(),
                },
            },
        }
        if err := raftNode.BootstrapCluster(conf).Error(); err != nil {
            return false, err
        }
    }

    n.raftNode = raftNode
    return hasState, nil
}

// Start 启动 HTTP 服务，提供对外接口。
func (n *Node) Start() error {
	addr := fmt.Sprintf(":%d", n.cfg.HTTPPort)
	log.Printf("node %s listening on %s (raft bind %s, data=%s, raft=%s)", n.cfg.NodeID, addr, n.cfg.RaftBind, n.cfg.DataDir, n.cfg.RaftDir)
	return n.server.ListenAndServe(addr)
}

// Close 关闭 Raft 和 Badger。
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

// proposeTransaction 将交易序列化后提交给 Raft 日志。
func (n *Node) proposeTransaction(tx *types.Transaction) error {
    if n.raftNode == nil {
        return errors.New("raft not initialized")
    }
    payload, err := json.Marshal(raftCommand{Type: commandTransaction, Transaction: tx})
    if err != nil {
        return err
    }
    future := n.raftNode.Apply(payload, 5*time.Second)
    if err := future.Error(); err != nil {
        return err
    }
    if respErr, ok := future.Response().(error); ok && respErr != nil {
        return respErr
    }
    return nil
}

// joinCluster 尝试联系集群节点完成加入操作。
func (n *Node) joinCluster() error {
    if len(n.cfg.RaftPeers) == 0 {
        return errors.New("raft_peers required for join")
    }
    body, _ := json.Marshal(joinRequest{NodeID: n.cfg.NodeID, RaftAddress: n.cfg.RaftBind})
    visited := map[string]bool{}
    queue := append([]string{}, n.cfg.RaftPeers...)
    client := &http.Client{Timeout: 5 * time.Second}
    for len(queue) > 0 {
        peer := queue[0]
        queue = queue[1:]
        if visited[peer] {
            continue
        }
        visited[peer] = true
        url := fmt.Sprintf("http://%s/raft/join", peer)
        resp, err := client.Post(url, "application/json", bytes.NewReader(body))
        if err != nil {
            log.Printf("join request to %s failed: %v", url, err)
            continue
        }
        if resp.StatusCode == http.StatusOK {
            resp.Body.Close()
            return nil
        }
        leader := resp.Header.Get("X-Raft-Leader")
        resp.Body.Close()
        if leader != "" && !visited[leader] {
            queue = append(queue, leader)
        }
    }
    return errors.New("failed to join raft cluster")
}

// handleJoinRequest 响应其它节点提交的 join 请求。
func (n *Node) handleJoinRequest(nodeID, raftAddr string) (string, error) {
    if n.raftNode == nil {
        return "", errors.New("raft not initialized")
    }
    if n.raftNode.State() != raft.Leader {
        leader := string(n.raftNode.Leader())
        if leader == "" {
            return "", errors.New("no leader")
        }
        return leader, errors.New("not leader")
    }
    future := n.raftNode.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(raftAddr), 0, 0)
    if err := future.Error(); err != nil {
        return "", err
    }
    return "", nil
}

// raftStatus 返回当前节点的 Raft 状态信息。
func (n *Node) raftStatus() map[string]interface{} {
    if n.raftNode == nil {
        return map[string]interface{}{"state": "not_initialized"}
    }
    stats := n.raftNode.Stats()
    return map[string]interface{}{
        "node_id":        n.cfg.NodeID,
        "state":          n.raftNode.State().String(),
        "leader":         string(n.raftNode.Leader()),
        "term":           stats["term"],
        "last_log_index": stats["last_log_index"],
        "applied_index":  stats["applied_index"],
    }
}

// raftCommand 为 Raft 日志条目的统一格式。
type raftCommand struct {
	Type        string             `json:"type"`
	Transaction *types.Transaction `json:"transaction,omitempty"`
}

// fsm 实现 raft.FSM 接口，负责真正的状态变更。
type fsm struct {
	txSvc *service.TransactionService
	db    *badger.DB
}

// Apply 会在日志提交后执行具体业务操作。
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

// Snapshot 使用 Badger 自带备份生成快照。
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	return &badgerSnapshot{db: f.db}, nil
}

// Restore 清空 Badger 并从快照恢复。
func (f *fsm) Restore(rc io.ReadCloser) error {
    defer rc.Close()
    if err := f.db.DropAll(); err != nil {
        return err
    }
    return f.db.Load(rc, 10)
}

// badgerSnapshot 负责将 Badger 快照写入 Raft sink。
type badgerSnapshot struct {
	db *badger.DB
}

func (s *badgerSnapshot) Persist(sink raft.SnapshotSink) error {
    if _, err := s.db.Backup(sink, 0); err != nil {
        sink.Cancel()
        return err
    }
    return sink.Close()
}

func (s *badgerSnapshot) Release() {}
