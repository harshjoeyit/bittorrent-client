package peer

import (
	"encoding/binary"
	"fmt"
	"log"
)

func chokeMsgHandler() {
	log.Println("CHOKE message received")
}

func unchokeMsgHandler() {
	log.Println("UNCHOKE message received")
}

func haveMsgHandler(payload []byte) {
	fmt.Println("HAVE message received")
	if len(payload) != 4 {
		log.Printf("payload for have message should be 4 bytes, got %d\n", payload)
	}

	// payload contains the piece index
	pieceIdx := binary.BigEndian.Uint32(payload)

	fmt.Println("HAVE: ", pieceIdx)
}

func bitfieldMsgHandler(payload []byte) []int {
	log.Println("BITFIELD message received", len(payload))

	var indices []int

	// Process byte by byte
	for i, val := range payload {
		adder := i * 8
		// Check all the 8 bits
		for i := 0; i < 8; i++ {
			// check if bit is set
			if (1<<i)&val > 0 {
				// index in current byte = 8 - i, since MSB denotes lower index
				idx := 7 - i
				indices = append(indices, adder+idx)
			}
		}
	}

	fmt.Println("BITFIELD decoded indices: ", indices)

	return indices
}

func pieceMsgHandler(payload []byte) {
	log.Println("PIECE message received", len(payload))
}
