package peer

import (
	cryptoRand "crypto/rand"
	"fmt"
	"my-bittorrent/peer/queue"
	"net"
)

// Peer represents a single node participating in a torrent network
type Peer struct {
	ID        [20]byte // Received in handshake response
	IPAddress net.IP
	Port      uint16
	Conn      net.Conn // TCP connection
	TaskQueue queue.Queue
}

func NewPeer(ip net.IP, port uint16) *Peer {
	return &Peer{
		IPAddress: ip,
		Port:      port,
		TaskQueue: queue.NewQueue(),
	}
}

var PeerID [20]byte

// GetPeerID generates and returns Peer ID for this client
func GetPeerID() ([20]byte, error) {
	// Check if the peerID has already been generated
	if PeerID != [20]byte{} {
		return PeerID, nil
	}

	// Client identifier (e.g., "AT" for Azureus-style)
	//
	// Azureus-style uses the following encoding: '-', two characters for
	// client id, four ascii digits for version number, '-', followed by
	// random numbers. For example: '-AT0001-'...
	clientID := "-AT0001-" // 8 bytes

	copy(PeerID[:], clientID)

	// Generate the remaining part randomly
	randomBytes := make([]byte, 12)
	_, err := cryptoRand.Read(randomBytes)
	if err != nil {
		return PeerID, fmt.Errorf("error in generating random bytes %v", err)
	}

	copy(PeerID[len(clientID):], randomBytes)

	return PeerID, nil
}

func GetCachedPeers() []*Peer {
	return []*Peer{
		{ID: [20]byte{}, IPAddress: net.ParseIP("49.37.249.9"), Port: 6881, Conn: nil},
	}
}
