package p2p

import (
	"context"
	"io"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/mr-tron/base58/base58"
)

// PeerID is a p2p peer identity.
type PeerID string

// Bytes returns the peer ID as byte slice
func (pid PeerID) Bytes() []byte {
	return []byte(pid)
}

// Pretty returns a b58-encoded string of the peer id
func (pid PeerID) Pretty() string {
	return base58.Encode(pid.Bytes())
}

// SendableData represents the struct used in data throttler implementation
type SendableData struct {
	Buff  []byte
	Topic string
}

// MessageProcessor is the interface used to describe what a receive message processor should do
// All implementations that will be called from Messenger implementation will need to satisfy this interface
// If the function returns a non nil value, the received message will not be propagated to its connected peers
type MessageProcessor interface {
	ProcessReceivedMessage(message MessageP2P, broadcastHandler func(buffToSend []byte)) error
	IsInterfaceNil() bool
}

// Messenger is the main struct used for communication with other peers
type Messenger interface {
	io.Closer

	// ID is the Messenger's unique peer identifier across the network (a
	// string). It is derived from the public key of the P2P credentials.
	ID() PeerID

	// Peers is the list of IDs of peers known to the Messenger.
	Peers() []PeerID

	// Addresses is the list of addresses that the Messenger is currently bound
	// to and listening to.
	Addresses() []string

	// ConnectedPeers returns the list of IDs of the peers the Messenger is
	// currently connected to.
	ConnectedPeers() []PeerID

	// ConnectToPeer explicitly connect to a specific peer with a known address (note that the
	// address contains the peer ID). This function is usually not called
	// manually, because any underlying implementation of the Messenger interface
	// should be keeping connections to peers open.
	ConnectToPeer(address string) error

	// Bootstrap runs the initialization phase which includes peer discovery,
	// setting up initial connections and self-announcement in the network.
	Bootstrap() error

	// CreateTopic defines a new topic for sending messages, and optionally
	// creates a channel in the LoadBalancer for this topic (otherwise, the topic
	// will use a default channel).
	CreateTopic(name string, createChannelForTopic bool) error

	// HasTopic returns true if the Messenger has declared interest in a topic
	// and it is listening to messages referencing it.
	HasTopic(name string) bool

	// HasTopicValidator returns true if the Messenger has registered a custom
	// validator for a given topic name.
	HasTopicValidator(name string) bool

	// RegisterMessageProcessor adds the provided MessageProcessor to the list
	// of handlers that are invoked whenever a message is received on the
	// specified topic.
	RegisterMessageProcessor(topic string, handler MessageProcessor) error

	// Broadcast is a convenience function that calls BroadcastOnChannelBlocking,
	// but implicitly sets the channel to be identical to the specified topic.
	Broadcast(topic string, buff []byte)

	// IsInterfaceNil returns true if there is no value under the interface
	IsInterfaceNil() bool
}

// MessageP2P defines what a p2p message can do (should return)
type MessageP2P interface {
	From() []byte
	Data() []byte
	SeqNo() []byte
	TopicIDs() []string
	Signature() []byte
	Key() []byte
	Peer() PeerID
	IsInterfaceNil() bool
}

// PeerDiscoverer defines the behaviour of a peer discovery mechanism
type PeerDiscoverer interface {
	Bootstrap() error
	Name() string

	ApplyContext(host host.Host, ctx context.Context) error
	IsInterfaceNil() bool
}
