package integrationTests

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/ElrondNetwork/elrond-go-p2p/p2p"
	"github.com/ElrondNetwork/elrond-go-p2p/p2p/discovery"
	"github.com/ElrondNetwork/elrond-go-p2p/p2p/libp2p"
	"github.com/btcsuite/btcd/btcec"
	libp2pCrypto "github.com/libp2p/go-libp2p-core/crypto"
)

// GetConnectableAddress returns a non circuit, non windows default connectable address for provided messenger
func GetConnectableAddress(mes p2p.Messenger) string {
	for _, addr := range mes.Addresses() {
		if strings.Contains(addr, "circuit") || strings.Contains(addr, "169.254") {
			continue
		}
		return addr
	}
	return ""
}

// CreateKadPeerDiscoverer creates a default kad peer dicoverer instance to be used in tests
func CreateKadPeerDiscoverer(peerRefreshInterval time.Duration, initialPeersList []string) p2p.PeerDiscoverer {
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: peerRefreshInterval,
		RandezVous:           "test",
		InitialPeersList:     initialPeersList,
		BucketSize:           100,
		RoutingTableRefresh:  time.Minute,
	}
	peerDiscovery, _ := discovery.NewKadDhtPeerDiscoverer(arg)

	return peerDiscovery
}

// CreateMessengerWithKadDht creates a new libp2p messenger with kad-dht peer discovery
func CreateMessengerWithKadDht(ctx context.Context, initialAddr string) p2p.Messenger {
	prvKey, _ := ecdsa.GenerateKey(btcec.S256(), rand.Reader)
	sk := (*libp2pCrypto.Secp256k1PrivateKey)(prvKey)

	libP2PMes, err := libp2p.NewNetworkMessengerOnFreePort(
		ctx,
		sk,
		nil,
		CreateKadPeerDiscoverer(time.Second*2, []string{initialAddr}),
	)
	if err != nil {
		fmt.Println(err.Error())
	}

	return libP2PMes
}

// WaitForBootstrapAndShowConnected will delay a given duration in order to wait for bootstraping  and print the
// number of peers that each node is connected to
func WaitForBootstrapAndShowConnected(peers []p2p.Messenger, durationBootstrapingTime time.Duration) {
	fmt.Printf("Waiting %v for peer discovery...\n", durationBootstrapingTime)
	time.Sleep(durationBootstrapingTime)

	fmt.Println("Connected peers:")
	for _, peer := range peers {
		fmt.Printf("Peer %s is connected to %d peers\n", peer.ID().Pretty(), len(peer.ConnectedPeers()))
	}
}
