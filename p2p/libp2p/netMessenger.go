package libp2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go-p2p/core/check"
	"github.com/ElrondNetwork/elrond-go-p2p/p2p"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/connmgr"
	libp2pCrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
)

const durationBetweenSends = time.Microsecond * 10
const pubsubTimeCacheDuration = 10 * time.Minute

// ListenAddrWithIp4AndTcp defines the listening address with ip v.4 and TCP
const ListenAddrWithIp4AndTcp = "/ip4/0.0.0.0/tcp/"

// ListenLocalhostAddrWithIp4AndTcp defines the local host listening ip v.4 address and TCP
const ListenLocalhostAddrWithIp4AndTcp = "/ip4/127.0.0.1/tcp/"

type networkMessenger struct {
	host           host.Host
	ctx            context.Context
	mutProcessors  sync.RWMutex
	topics         map[string]*pubsub.Topic
	processors     map[string]p2p.MessageProcessor
	pb             *pubsub.PubSub
	publishChan    chan *p2p.SendableData
	peerDiscoverer p2p.PeerDiscoverer
}

// NewNetworkMessenger creates a libP2P messenger by opening a port on the current machine
// Should be used in production!
func NewNetworkMessenger(
	ctx context.Context,
	port int,
	p2pPrivKey libp2pCrypto.PrivKey,
	conMgr connmgr.ConnManager,
	_ interface{},
	peerDiscoverer p2p.PeerDiscoverer,
	listenAddress string,
	_ int,
) (*networkMessenger, error) {
	address := fmt.Sprintf(listenAddress+"%d", port)
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(address),
		libp2p.Identity(p2pPrivKey),
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.ConnectionManager(conMgr),
		libp2p.DefaultTransports,
		//we need the disable relay option in order to save the node's bandwidth as much as possible
		libp2p.DisableRelay(),
		libp2p.NATPortMap(),
	}

	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	p2pNode, err := createMessenger(h, ctx, true, peerDiscoverer)
	if err != nil {
		fmt.Println(err)

		return nil, err
	}

	return p2pNode, nil
}

func createMessenger(
	host host.Host,
	ctx context.Context,
	withSigning bool,
	peerDiscoverer p2p.PeerDiscoverer,
) (*networkMessenger, error) {

	pb, err := createPubSub(host, ctx, withSigning)
	if err != nil {
		return nil, err
	}

	err = peerDiscoverer.ApplyContext(host, ctx)
	if err != nil {
		return nil, err
	}

	netMes := networkMessenger{
		host:           host,
		ctx:            ctx,
		pb:             pb,
		processors:     make(map[string]p2p.MessageProcessor),
		topics:         make(map[string]*pubsub.Topic),
		publishChan:    make(chan *p2p.SendableData),
		peerDiscoverer: peerDiscoverer,
	}

	go func(pubsub *pubsub.PubSub, publishChan chan *p2p.SendableData) {
		for {
			sendableData := <-publishChan

			if sendableData == nil {
				continue
			}

			netMes.mutProcessors.RLock()
			topic := netMes.topics[sendableData.Topic]
			netMes.mutProcessors.RUnlock()

			if topic != nil {
				err := topic.Publish(context.Background(), sendableData.Buff)
				if err != nil {
					fmt.Printf("error publishing: %s\n", err.Error())
				}
			} else {
				fmt.Printf("error publishing: not joined on topic %s\n", sendableData.Topic)
			}

			time.Sleep(durationBetweenSends)
		}
	}(pb, netMes.publishChan)

	fmt.Println("listening on addresses:")
	for i, address := range netMes.host.Addrs() {
		fmt.Printf("   addr%d: %s\n", i, address.String()+"/p2p/"+netMes.ID().Pretty())
	}

	return &netMes, nil
}

func createPubSub(host host.Host, ctx context.Context, withSigning bool) (*pubsub.PubSub, error) {
	optsPS := []pubsub.Option{
		pubsub.WithMessageSigning(withSigning),
	}

	pubsub.TimeCacheDuration = pubsubTimeCacheDuration

	ps, err := pubsub.NewGossipSub(ctx, host, optsPS...)
	if err != nil {
		return nil, err
	}

	return ps, nil
}

// NewNetworkMessengerOnFreePort tries to create a new NetworkMessenger on a free port found in the system
// Should be used only in testing!
func NewNetworkMessengerOnFreePort(
	ctx context.Context,
	p2pPrivKey libp2pCrypto.PrivKey,
	conMgr connmgr.ConnManager,
	peerDiscoverer p2p.PeerDiscoverer,
) (*networkMessenger, error) {
	return NewNetworkMessenger(
		ctx,
		0,
		p2pPrivKey,
		conMgr,
		nil,
		peerDiscoverer,
		ListenLocalhostAddrWithIp4AndTcp,
		0,
	)
}

// Close closes the host, connections and streams
func (netMes *networkMessenger) Close() error {
	return netMes.host.Close()
}

// ID returns the messenger's ID
func (netMes *networkMessenger) ID() p2p.PeerID {
	return p2p.PeerID(netMes.host.ID())
}

// Addresses returns all addresses found in peerstore
func (netMes *networkMessenger) Addresses() []string {
	addrs := make([]string, 0)

	for _, address := range netMes.host.Addrs() {
		addrs = append(addrs, address.String()+"/p2p/"+netMes.ID().Pretty())
	}

	return addrs
}

// ConnectToPeer tries to open a new connection to a peer
func (netMes *networkMessenger) ConnectToPeer(address string) error {
	return ConnectToPeer(netMes.host, netMes.ctx, address)
}

// ConnectedPeers returns the current connected peers list
func (netMes *networkMessenger) ConnectedPeers() []p2p.PeerID {
	connectedPeers := make(map[p2p.PeerID]struct{})

	for _, conn := range netMes.host.Network().Conns() {
		p := p2p.PeerID(conn.RemotePeer())

		if netMes.isConnected(p) {
			connectedPeers[p] = struct{}{}
		}
	}

	peerList := make([]p2p.PeerID, len(connectedPeers))

	index := 0
	for k := range connectedPeers {
		peerList[index] = k
		index++
	}

	return peerList
}

// isConnected returns true if current node is connected to provided peer
func (netMes *networkMessenger) isConnected(peerID p2p.PeerID) bool {
	connectedness := netMes.host.Network().Connectedness(peer.ID(peerID))

	return connectedness == network.Connected
}

// Peers returns the list of all known peers ID (including self)
func (netMes *networkMessenger) Peers() []p2p.PeerID {
	h := netMes.host
	peers := make([]p2p.PeerID, 0)

	for _, p := range h.Peerstore().Peers() {
		peers = append(peers, p2p.PeerID(p))
	}
	return peers
}

// Bootstrap will start the peer discovery mechanism
func (netMes *networkMessenger) Bootstrap() error {
	return netMes.peerDiscoverer.Bootstrap()
}

// CreateTopic opens a new topic using pubsub infrastructure
func (netMes *networkMessenger) CreateTopic(name string, _ bool) error {
	netMes.mutProcessors.Lock()
	defer netMes.mutProcessors.Unlock()
	_, found := netMes.topics[name]
	if found {
		return p2p.ErrTopicAlreadyExists
	}

	netMes.processors[name] = nil
	topic, err := netMes.pb.Join(name)
	if err != nil {
		return err
	}

	netMes.topics[name] = topic

	subscrRequest, err := topic.Subscribe()
	if err != nil {
		return err
	}

	//just a dummy func to consume messages received by the newly created topic
	go func() {
		for {
			_, _ = subscrRequest.Next(netMes.ctx)
		}
	}()

	return err
}

// HasTopic returns true if the topic has been created
func (netMes *networkMessenger) HasTopic(name string) bool {
	netMes.mutProcessors.RLock()
	_, found := netMes.topics[name]
	netMes.mutProcessors.RUnlock()

	return found
}

// HasTopicValidator returns true if the topic has a validator set
func (netMes *networkMessenger) HasTopicValidator(name string) bool {
	netMes.mutProcessors.RLock()
	validator := netMes.processors[name]
	netMes.mutProcessors.RUnlock()

	return validator != nil
}

// RegisterMessageProcessor registers a message process on a topic
func (netMes *networkMessenger) RegisterMessageProcessor(topic string, handler p2p.MessageProcessor) error {
	if check.IfNil(handler) {
		return p2p.ErrNilValidator
	}

	netMes.mutProcessors.Lock()
	defer netMes.mutProcessors.Unlock()
	validator, found := netMes.processors[topic]
	if !found {
		return p2p.ErrNilTopic
	}
	if validator != nil {
		return p2p.ErrTopicValidatorOperationNotSupported
	}

	broadcastHandler := func(buffToSend []byte) {
		netMes.Broadcast(topic, buffToSend)
	}

	err := netMes.pb.RegisterTopicValidator(topic, func(ctx context.Context, pid peer.ID, message *pubsub.Message) bool {
		err := handler.ProcessReceivedMessage(NewMessage(message), broadcastHandler)
		if err != nil {
			fmt.Printf("p2p validator error %s on topics %v\n", err.Error(), message.TopicIDs)
		}

		return err == nil
	})
	if err != nil {
		return err
	}

	netMes.processors[topic] = handler
	return nil
}

// Broadcast tries to send a byte buffer onto a topic. Not a blocking call
func (netMes *networkMessenger) Broadcast(topic string, buff []byte) {
	go func(t string, b []byte) {
		sendable := &p2p.SendableData{
			Buff:  b,
			Topic: t,
		}

		netMes.publishChan <- sendable
	}(topic, buff)
}

// IsInterfaceNil returns true if there is no value under the interface
func (netMes *networkMessenger) IsInterfaceNil() bool {
	return netMes == nil
}

// ConnectToPeer connects the host to a provided address
func ConnectToPeer(host host.Host, ctx context.Context, address string) error {
	multiAddr, err := multiaddr.NewMultiaddr(address)
	if err != nil {
		return err
	}

	pInfo, err := peer.AddrInfoFromP2pAddr(multiAddr)
	if err != nil {
		return err
	}

	return host.Connect(ctx, *pInfo)
}
