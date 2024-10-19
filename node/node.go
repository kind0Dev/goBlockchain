package node

import (
	"context"
	"encoding/hex"
	"net"
	"sync"
	"time"

	"github.com/djeday123/blockchain2/crypto"
	"github.com/djeday123/blockchain2/proto"
	"github.com/djeday123/blockchain2/types"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	//"google.golang.org/grpc/credentials/insecure"
	_ "github.com/davecgh/go-spew/spew"
	_ "github.com/gookit/goutil/dump"
	"google.golang.org/grpc/peer"
)

// type Server struct {
// 	listenAddr string
// 	ln         net.Listener
// }

// func New(listenAddr string) (*Server, error) {
// 	ln, err := net.Listen("tpc", listenAddr)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &Server{
// 		listenAddr: listenAddr,
// 		ln:         ln,
// 	}, nil
// }

const blockTime = time.Second * 5

type Mempool struct {
	lock sync.RWMutex
	txx  map[string]*proto.Transaction
}

func NewMemPool() *Mempool {
	return &Mempool{
		txx: make(map[string]*proto.Transaction),
	}
}

func (pool *Mempool) Clear() []*proto.Transaction {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	txx := make([]*proto.Transaction, len(pool.txx))
	it := 0
	for k, v := range pool.txx {
		delete(pool.txx, k)
		txx[it] = v
		it++
	}
	return txx
}

func (pool *Mempool) Len() int {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	return len(pool.txx)
}

func (pool *Mempool) Has(tx *proto.Transaction) bool {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	hash := hex.EncodeToString(types.HashTransaction(tx))
	_, ok := pool.txx[hash]
	return ok
}

func (pool *Mempool) Add(tx *proto.Transaction) bool {
	if pool.Has(tx) {
		return false
	}

	pool.lock.Lock()
	defer pool.lock.Unlock()

	hash := hex.EncodeToString(types.HashTransaction(tx))
	pool.txx[hash] = tx
	return true
}

type ServerConfig struct {
	Version    string
	ListenAddr string
	PrivateKey *crypto.PrivateKey
}

type Node struct {
	//peers map[net.Addr]*grpc.ClientConn
	ServerConfig
	logger *zap.SugaredLogger

	peerLock sync.RWMutex
	peers    map[proto.NodeClient]*proto.Version
	mempool  *Mempool

	proto.UnimplementedNodeServer
}

func NewNode(cfg ServerConfig) *Node {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.Development = true
	loggerConfig.EncoderConfig.TimeKey = ""
	//loggerConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
	logger, _ := loggerConfig.Build()
	return &Node{
		peers:        make(map[proto.NodeClient]*proto.Version),
		logger:       logger.Sugar(),
		mempool:      NewMemPool(),
		ServerConfig: cfg,
	}
}

func (n *Node) Start(listenAddr string, bootstrapNodes []string) error {
	n.ListenAddr = listenAddr

	var (
		opts       = []grpc.ServerOption{}
		grpcServer = grpc.NewServer(opts...)
	)

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	proto.RegisterNodeServer(grpcServer, n)

	n.logger.Infow("node started...", "port", n.ListenAddr)
	// bootstrap the network with list of already known hodes in the network
	if len(bootstrapNodes) > 0 {
		go n.bootstrapNetwork(bootstrapNodes)
	}

	if n.PrivateKey != nil {
		go n.validatorLoop()
	}
	return grpcServer.Serve(ln)
}

func (n *Node) Handshake(ctx context.Context, v *proto.Version) (*proto.Version, error) {
	c, err := makeNodeClient(v.ListenAddr)
	if err != nil {
		return nil, err
	}
	n.addPeer(c, v)

	return n.getVersion(), nil
}

func (n *Node) HandleTransaction(ctx context.Context, tx *proto.Transaction) (*proto.Ack, error) {
	peer, _ := peer.FromContext(ctx)
	hash := hex.EncodeToString(types.HashTransaction(tx))

	if n.mempool.Add(tx) {
		n.logger.Debugw("received tx", "from", peer.Addr, "hash", hash, "we", n.ListenAddr)
		go func() {
			if err := n.broadcast(tx); err != nil {
				n.logger.Errorw("broadcast error", "err", err)
			}
		}()
	}

	return &proto.Ack{}, nil
}

func (n *Node) validatorLoop() {

	n.logger.Infow("starting validator loop", "pubkey", n.PrivateKey.Public(), "blockTime", blockTime)
	ticker := time.NewTicker(blockTime)
	for {
		<-ticker.C

		n.logger.Debugw("time to create a new block 1 ", "lenTx", len(n.mempool.txx))
		txx := n.mempool.Clear()
		n.logger.Debugw("time to create a new block 2", "lenTx", len(txx))
	}
}

func (n *Node) broadcast(msg any) error {
	for peer := range n.peers {
		switch v := msg.(type) {
		case *proto.Transaction:
			//n.logger.Debugw("broadcast", "node", n.listenAddr, "from", peer)
			_, err := peer.HandleTransaction(context.Background(), v)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (n *Node) addPeer(c proto.NodeClient, v *proto.Version) {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()

	// Handle the logic where we decide to accepr or drop the incoming node connection

	n.peers[c] = v
	if len(v.PeerList) > 0 {
		go n.bootstrapNetwork(v.PeerList)
	}
	// Connect to all peers in the received list of peers

	n.logger.Debugw("new peer successfully connected",
		"we", n.ListenAddr,
		"remoteNode", v.ListenAddr,
		"height", v.Height)
}

func (n *Node) deletePeer(c proto.NodeClient) {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()
	delete(n.peers, c)
}

func (n *Node) bootstrapNetwork(addrs []string) error {
	for _, addr := range addrs {
		if !n.canConnectWith(addr) {
			continue
		}
		n.logger.Debugw("dialing remote node", "we", n.ListenAddr, "remote", addr)

		c, v, err := n.dialRemoteNode(addr)
		if err != nil {
			return err
		}
		n.addPeer(c, v)
	}

	return nil
}

func (n *Node) dialRemoteNode(addr string) (proto.NodeClient, *proto.Version, error) {
	c, err := makeNodeClient(addr)
	if err != nil {
		return nil, nil, err
	}

	v, err := c.Handshake(context.Background(), n.getVersion())
	if err != nil {
		n.logger.Error("handshake error:", err)
		return nil, nil, err
	}

	return c, v, nil
}

func (n *Node) getVersion() *proto.Version {
	return &proto.Version{
		Version:    "bl-0.1",
		Height:     0,
		ListenAddr: n.ListenAddr,
		PeerList:   n.getPeerList(),
	}
}

func (n *Node) canConnectWith(addr string) bool {
	if n.ListenAddr == addr {
		return false
	}
	connectedPeers := n.getPeerList()
	for _, connectedAddr := range connectedPeers {
		if addr == connectedAddr {
			return false
		}
	}
	return true
}

func (n *Node) getPeerList() []string {
	n.peerLock.RLock()
	defer n.peerLock.RUnlock()

	peers := []string{}
	for _, version := range n.peers {
		peers = append(peers, version.ListenAddr)
	}
	return peers
}

func makeNodeClient(listenAddr string) (proto.NodeClient, error) {
	c, err := grpc.Dial(listenAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return proto.NewNodeClient(c), nil
}
