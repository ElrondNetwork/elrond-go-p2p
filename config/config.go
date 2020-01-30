package config

// KadDhtPeerDiscoveryConfig will hold the kad-dht discovery config settings
type KadDhtPeerDiscoveryConfig struct {
	Enabled                          bool
	RefreshIntervalInSec             int
	RandezVous                       string
	InitialPeerList                  []string
	BucketSize                       uint32
	RoutingTableRefreshIntervalInSec uint32
}

// P2PConfig will hold all the P2P settings
type P2PConfig struct {
	Node                NodeConfig
	KadDhtPeerDiscovery KadDhtPeerDiscoveryConfig
}

// NodeConfig will hold basic p2p settings
type NodeConfig struct {
	Port            int
	Seed            string
	TargetPeerCount int
}
