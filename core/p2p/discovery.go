package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus" // Using logrus for structured logging
)

// PeerInfo stores information about a connected peer
type PeerInfo struct {
	ID        peer.ID
	Addresses []multiaddr.Multiaddr
	LastSeen  time.Time
}

// Discovery handles peer discovery
type Discovery struct {
	host     host.Host
	dht      *dht.IpfsDHT
	mdns     mdns.Service
	network  *P2PNetwork
	peerInfo map[peer.ID]*PeerInfo
	security *PeerSecurity
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	wg       sync.WaitGroup
}

// NewDiscovery creates a new discovery service
func NewDiscovery(network *P2PNetwork) *Discovery {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Initialize security with default configuration
	secConfig := SecurityConfig{
		MaxPeersPerMinute:  30,
		MinPeerScore:       0.5,
		MaxPeersPerRegion:  10,
	}
	
	return &Discovery{
		host:     network.host,
		network:  network,
		peerInfo: make(map[peer.ID]*PeerInfo),
		security: NewPeerSecurity(secConfig),
		ctx:      ctx,
		cancel:   cancel,
		done:     make(chan struct{}),
	}
}

// Start starts the discovery service
func (d *Discovery) Start(ctx context.Context) error {
	// Initialize DHT
	dht, err := dht.New(ctx, d.host)
	if err != nil {
		return fmt.Errorf("failed to create DHT: %w", err)
	}
	d.dht = dht

	if err := d.dht.Bootstrap(ctx); err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// Initialize mDNS
	d.mdns = mdns.NewMdnsService(d.host, "timeserver", d)

	// Start peer discovery
	d.wg.Add(1)
	go d.discoverPeers()

	return nil
}

// Stop stops the discovery service
func (d *Discovery) Stop() error {
	close(d.done)
	d.cancel()
	d.wg.Wait()

	if err := d.dht.Close(); err != nil {
		return fmt.Errorf("failed to close DHT: %w", err)
	}

	return nil
}

// HandlePeerFound implements mdns.Notifee
func (d *Discovery) HandlePeerFound(pi peer.AddrInfo) {
	d.AddPeer(pi.ID, pi.Addrs)
}

// AddPeer adds a peer to the discovery service
func (d *Discovery) AddPeer(p peer.ID, addrs []multiaddr.Multiaddr) {
	// Validate peer before adding
	if !d.security.ValidateNewPeer(p) {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.peerInfo[p] = &PeerInfo{
		ID:        p,
		Addresses: addrs,
		LastSeen:  time.Now(),
	}

	// Add to peerstore
	for _, addr := range addrs {
		d.host.Peerstore().AddAddr(p, addr, peerstore.PermanentAddrTTL)
	}

	// Check for potential Sybil attack
	peers := make([]peer.ID, 0, len(d.peerInfo))
	for id := range d.peerInfo {
		peers = append(peers, id)
	}
	if d.security.DetectSybilAttack(peers) {
		// Log the suspicious activity but don't disconnect immediately
		// This allows for monitoring while maintaining network functionality
		logrus.Warn("Potential Sybil attack detected")
	}
}

// RemovePeer removes a peer from the discovery service
func (d *Discovery) RemovePeer(p peer.ID) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.peerInfo, p)
}

// GetPeers returns all known peers
func (d *Discovery) GetPeers() []*PeerInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	peers := make([]*PeerInfo, 0, len(d.peerInfo))
	for _, info := range d.peerInfo {
		peers = append(peers, info)
	}
	return peers
}

// discoverPeers continuously discovers new peers
func (d *Discovery) discoverPeers() {
	defer d.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			peers := d.dht.RoutingTable().ListPeers()
			for _, p := range peers {
				// Get peer addresses
				addrs := d.host.Peerstore().Addrs(p)
				if len(addrs) > 0 {
					// Only add peer if it passes security checks
					if d.security.ValidateNewPeer(p) {
						d.AddPeer(p, addrs)
					}
				}
			}

			// Update peer scores and remove low-scoring peers
			d.updatePeerScores()
			d.removeStalePeers()
		case <-d.done:
			return
		}
	}
}

// updatePeerScores updates the scores of all peers
func (d *Discovery) updatePeerScores() {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for id, info := range d.peerInfo {
		// Calculate time drift and response time
		// This is a simplified example - in practice, you would use actual measurements
		timeDrift := time.Duration(0)
		responseTime := time.Since(info.LastSeen)
		
		d.security.UpdatePeerScore(id, timeDrift, responseTime)
		
		// Disconnect peers with low scores
		if d.security.ShouldDisconnectPeer(id) {
			d.RemovePeer(id)
			d.host.Network().ClosePeer(id)
		}
	}
}

// removeStalePeers removes peers that haven't been seen recently
func (d *Discovery) removeStalePeers() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	for id, info := range d.peerInfo {
		if now.Sub(info.LastSeen) > 5*time.Minute {
			delete(d.peerInfo, id)
		}
	}
}
