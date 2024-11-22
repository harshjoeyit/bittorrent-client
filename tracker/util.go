package tracker

import (
	cryptoRand "crypto/rand"
	"fmt"
	"math/rand"
	"time"
)

var PeerID [20]byte

func randInt32() int32 {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	return r.Int31()
}

func getPeerID() ([20]byte, error) {
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
