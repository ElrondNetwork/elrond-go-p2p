package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go-p2p/p2p"
	"github.com/ElrondNetwork/elrond-go-p2p/p2p/libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	opts "github.com/libp2p/go-libp2p-kad-dht/opts"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
)

const (
	kadDhtName = "kad-dht discovery"
)

var peerDiscoveryTimeout = 10 * time.Second
var noOfQueries = 1

// ArgKadDht represents the kad-dht config argument DTO
type ArgKadDht struct {
	PeersRefreshInterval time.Duration
	RandezVous           string
	InitialPeersList     []string
	BucketSize           uint32
	RoutingTableRefresh  time.Duration
}

// KadDhtDiscoverer is the kad-dht discovery type implementation
type KadDhtDiscoverer struct {
	mutKadDht            sync.Mutex
	kadDHT               *dht.IpfsDHT
	host                 host.Host
	ctx                  context.Context
	peersRefreshInterval time.Duration
	randezVous           string
	initialPeersList     []string
	routingTableRefresh  time.Duration
	bucketSize           uint32
}

// NewKadDhtPeerDiscoverer creates a new kad-dht discovery type implementation
// initialPeersList can be nil or empty, no initial connection will be attempted, a warning message will appear
func NewKadDhtPeerDiscoverer(arg ArgKadDht) (*KadDhtDiscoverer, error) {
	if arg.PeersRefreshInterval < time.Second {
		return nil, fmt.Errorf("%w, PeersRefreshInterval should have been at least 1 second", p2p.ErrInvalidValue)
	}
	if arg.RoutingTableRefresh < time.Second {
		return nil, fmt.Errorf("%w, RoutingTableRefresh should have been at least 1 second", p2p.ErrInvalidValue)
	}
	isListNilOrEmpty := len(arg.InitialPeersList) == 0
	if isListNilOrEmpty {
		fmt.Println("nil or empty initial peers list provided to kad dht implementation. " +
			"No initial connection will be done")
	}

	return &KadDhtDiscoverer{
		peersRefreshInterval: arg.PeersRefreshInterval,
		randezVous:           arg.RandezVous,
		initialPeersList:     arg.InitialPeersList,
		bucketSize:           arg.BucketSize,
		routingTableRefresh:  arg.RoutingTableRefresh,
	}, nil
}

// Bootstrap will start the bootstrapping new peers process
func (kdd *KadDhtDiscoverer) Bootstrap() error {
	kdd.mutKadDht.Lock()
	defer kdd.mutKadDht.Unlock()

	if kdd.kadDHT != nil {
		return p2p.ErrPeerDiscoveryProcessAlreadyStarted
	}
	if kdd.host == nil {
		return fmt.Errorf("%w for host", p2p.ErrInvalidValue)
	}
	if kdd.ctx == nil {
		return fmt.Errorf("%w for context", p2p.ErrInvalidValue)
	}

	defaultOptions := opts.Defaults
	customOptions := func(opt *opts.Options) error {
		err := defaultOptions(opt)
		if err != nil {
			return err
		}

		return nil
	}

	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping node of the DHT can go down without
	// inhibiting future peer discovery.
	kademliaDHT, err := dht.New(kdd.ctx, kdd.host, customOptions)
	if err != nil {
		return err
	}

	go kdd.connectToInitialAndBootstrap()

	kdd.kadDHT = kademliaDHT
	return nil
}

func (kdd *KadDhtDiscoverer) connectToInitialAndBootstrap() {
	chanStartBootstrap := kdd.connectToOnePeerFromInitialPeersList(
		kdd.peersRefreshInterval,
		kdd.initialPeersList,
	)

	cfg := dht.BootstrapConfig{
		Period:  kdd.peersRefreshInterval,
		Queries: noOfQueries,
		Timeout: peerDiscoveryTimeout,
	}

	go func() {
		<-chanStartBootstrap

		kdd.mutKadDht.Lock()
		go func() {
			for {
				err := kdd.kadDHT.Bootstrap(kdd.ctx)
				if err == kbucket.ErrLookupFailure {
					<-kdd.ReconnectToNetwork()
				}

				select {
				case <-time.After(kdd.peersRefreshInterval):
				case <-kdd.ctx.Done():
					return
				}
			}
		}()
		kdd.mutKadDht.Unlock()
	}()
}

func (kdd *KadDhtDiscoverer) connectToOnePeerFromInitialPeersList(
	intervalBetweenAttempts time.Duration,
	initialPeersList []string) <-chan struct{} {

	chanDone := make(chan struct{}, 1)

	if initialPeersList == nil {
		chanDone <- struct{}{}
		return chanDone
	}

	if len(initialPeersList) == 0 {
		chanDone <- struct{}{}
		return chanDone
	}

	go func() {
		startIndex := 0

		for {
			err := libp2p.ConnectToPeer(kdd.host, kdd.ctx, initialPeersList[startIndex])

			if err != nil {
				//could not connect, wait and try next one
				startIndex++
				startIndex %= len(initialPeersList)

				time.Sleep(intervalBetweenAttempts)

				continue
			}

			chanDone <- struct{}{}
			return
		}
	}()

	return chanDone
}

// Name returns the name of the kad dht peer discovery implementation
func (kdd *KadDhtDiscoverer) Name() string {
	return kadDhtName
}

// ApplyContext sets the context in which this discoverer is to be run
func (kdd *KadDhtDiscoverer) ApplyContext(host host.Host, ctx context.Context) error {
	if host == nil {
		return fmt.Errorf("%w for host", p2p.ErrInvalidValue)
	}
	if ctx == nil {
		return fmt.Errorf("%w for context", p2p.ErrInvalidValue)
	}

	kdd.host = host
	kdd.ctx = ctx

	return nil
}

// ReconnectToNetwork will try to connect to one peer from the initial peer list
func (kdd *KadDhtDiscoverer) ReconnectToNetwork() <-chan struct{} {
	return kdd.connectToOnePeerFromInitialPeersList(
		kdd.peersRefreshInterval,
		kdd.initialPeersList,
	)
}

// IsInterfaceNil returns true if there is no value under the interface
func (kdd *KadDhtDiscoverer) IsInterfaceNil() bool {
	return kdd == nil
}
