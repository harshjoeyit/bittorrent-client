package peer

import (
	cryptoRand "crypto/rand"
	"fmt"
	"net"
)

// Peer represents a single node participating in a torrent network
type Peer struct {
	ID        [20]byte // Received in handshake response
	IPAddress net.IP
	Port      uint16
	Conn      net.Conn // TCP connection
	HasPieces []bool   // Pieces that a peer has
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
		{ID: [20]byte{}, IPAddress: net.ParseIP("49.37.249.9"), Port: 6881, Conn: nil, HasPieces: []bool{}},
		{ID: [20]byte{}, IPAddress: net.ParseIP("78.92.207.147"), Port: 42069, Conn: nil, HasPieces: []bool{}},
		{ID: [20]byte{}, IPAddress: net.ParseIP("146.70.107.220"), Port: 42069, Conn: nil, HasPieces: []bool{}},
		{ID: [20]byte{}, IPAddress: net.ParseIP("157.157.43.209"), Port: 42069, Conn: nil, HasPieces: []bool{}},
		{ID: [20]byte{}, IPAddress: net.ParseIP("220.246.210.35"), Port: 6881, Conn: nil, HasPieces: []bool{}},
		{ID: [20]byte{}, IPAddress: net.ParseIP("212.102.35.101"), Port: 39852, Conn: nil, HasPieces: []bool{}},
		{ID: [20]byte{}, IPAddress: net.ParseIP("212.92.104.216"), Port: 45671, Conn: nil, HasPieces: []bool{}},
		{ID: [20]byte{}, IPAddress: net.ParseIP("212.32.253.225"), Port: 21188, Conn: nil, HasPieces: []bool{}},
		{ID: [20]byte{}, IPAddress: net.ParseIP("197.232.29.6"), Port: 47704, Conn: nil, HasPieces: []bool{}},
		{ID: [20]byte{}, IPAddress: net.ParseIP("194.36.110.131"), Port: 63943, Conn: nil, HasPieces: []bool{}},
	}
}
