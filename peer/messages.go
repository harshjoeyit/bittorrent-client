package peer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

// messageID defines message IDs for peer to peer message exchange
type messageID byte

const (
	Choke         messageID = 0
	Unchoke       messageID = 1
	Interested    messageID = 2
	NotInterested messageID = 3
	Have          messageID = 4
	Bitfield      messageID = 5
	Request       messageID = 6
	Piece         messageID = 7
	Cancel        messageID = 8
	Port          messageID = 9
)

const ProtocolIdentifier string = "BitTorrent protocol"

type Message struct {
	ID      messageID
	Payload []byte
}

func BuildHandshakeMessage(infoHash [20]byte) ([]byte, error) {
	// handshake: <pstrlen><pstr><reserved><info_hash><peer_id>
	// bytes: 1 + pstrlen + 8 + 20 + 20 = (49 + pstrlen) bytes
	pstrlen := len(ProtocolIdentifier)
	msg := make([]byte, 49+pstrlen)

	// Add pstrlen as single byte
	msg[0] = byte(pstrlen)
	// Add pstr (protocol identifier)
	copy(msg[1:1+pstrlen], []byte(ProtocolIdentifier))
	// Add reserved (8 bytes, all zeroes)
	copy(msg[1+pstrlen:1+pstrlen+8], make([]byte, 8))
	// Add info_hash
	copy(msg[1+pstrlen+8:1+pstrlen+8+20], infoHash[:])
	// Add peer ID
	fmt.Println("peer ID: ", PeerID)
	copy(msg[1+pstrlen+8+20:], PeerID[:])

	fmt.Printf("handshake msg in hex: %x\n", msg)

	return msg, nil
}

func BuildKeepAliveMessage() []byte {
	return make([]byte, 4)
}

func BuildChokeMessage() []byte {
	// <len=0001><id=0>
	// bytes: 4 + 1 = 5
	msg := make([]byte, 5)
	var msgLen, msgID int = 1, int(Choke)

	copy(msg[0:4], intToBytes(msgLen, 4))
	copy(msg[4:5], intToBytes(msgID, 1))

	fmt.Printf("choke msg in hex: %x", msg)

	return msg
}

func BuildUnchokeMessage() []byte {
	// <len=0001><id=1>
	// bytes: 4 + 1 = 5
	msg := make([]byte, 5)
	var msgLen, msgID int = 1, int(Unchoke)

	copy(msg[0:4], intToBytes(msgLen, 4))
	copy(msg[4:5], intToBytes(msgID, 1))

	fmt.Printf("un-choke msg in hex: %x", msg)

	return msg
}

func BuildInterestedMessage() []byte {
	// <len=0001><id=2>
	// bytes: 4 + 1 = 5
	msg := make([]byte, 5)
	var msgLen, msgID int = 1, int(Interested)

	copy(msg[0:4], intToBytes(msgLen, 4))
	copy(msg[4:5], intToBytes(msgID, 1))

	fmt.Printf("interested msg in hex: %x", msg)

	return msg
}

func BuildNotInterestedMessage() []byte {
	// <len=0001><id=2>
	// bytes: 4 + 1 = 5
	msg := make([]byte, 5)
	var msgLen, msgID int = 1, int(NotInterested)

	copy(msg[0:4], intToBytes(msgLen, 4))
	copy(msg[4:5], intToBytes(msgID, 1))

	fmt.Printf("not interested msg in hex: %x", msg)

	return msg
}

func BuildHaveMessage(pieceIdx int) []byte {
	// <len=0005><id=4><piece index>
	// bytes: 4 + 1 + 4 = 9
	msg := make([]byte, 9)
	var msgLen, msgID int = 1, int(Have)

	copy(msg[0:4], intToBytes(msgLen, 4))
	copy(msg[4:5], intToBytes(msgID, 1))
	copy(msg[5:9], intToBytes(pieceIdx, 4))

	fmt.Printf("have msg in hex: %x", msg)

	return msg
}

func BuildBitFieldMessage(pieces []bool) ([]byte, error) {
	// <len=0001+X><id=5><bitfield>, where X is bytes required for bitfield
	// bytes: 4 + 1 + X

	// MSB refers to the lowest index i.e. 0
	// Calculate bytes needed to represent bitfield
	// Ecxample - If peices = 4, bitfield can be reperesented using 1 bytes
	// If we have pieces 0, 2, then bitfield will look like
	// [1 0 1 0 0 0 0 0]
	// Example - If pieces = 10, bitfield can be reperesented using 2 bytes
	// If we have pieces 3, 5, 8, 9
	// [0 0 0 1 0 1 0 0] [1 1 0 0 0 0 0 0]
	bytesForBitfield := (len(pieces) + 7) / 8 // Round up to nearest byte

	// Create the bitfield
	bitfield := make([]byte, bytesForBitfield)

	for i, hasPiece := range pieces {
		if hasPiece {
			byteIdx := i / 8
			bitIdx := 7 - (i % 8) // High bit corresponds to piece 0
			bitfield[byteIdx] |= (1 << bitIdx)
		}
	}

	buf := new(bytes.Buffer)

	// messsage length (4 bytes)
	msgLen := 1 + len(bitfield)
	if err := binary.Write(buf, binary.BigEndian, uint32(msgLen)); err != nil {
		return nil, fmt.Errorf("failed to write message length: %v", err)
	}

	// message ID (1 byte)
	msgID := intToBytes(int(Bitfield), 1)
	if _, err := buf.Write(msgID); err != nil {
		return nil, fmt.Errorf("failed to write message ID: %v", err)
	}

	// Write the bitfield (bytesForBitfield)
	if _, err := buf.Write(bitfield); err != nil {
		return nil, fmt.Errorf("failed to write bitfield: %v", err)
	}

	return buf.Bytes(), nil
}

func BuildRequestMessage(pieceIdx, blockOffset, reqLen int) []byte {
	// <len=0013><id=6><index><begin><length>
	// bytes: 4 + 1 + 4 + 4 + 4 = 17
	msg := make([]byte, 17)
	var msgLen, msgID int = 13, int(Request)

	copy(msg[0:4], intToBytes(msgLen, 4))
	copy(msg[4:5], intToBytes(msgID, 1))
	copy(msg[5:9], intToBytes(pieceIdx, 4))
	copy(msg[9:13], intToBytes(blockOffset, 4))
	copy(msg[13:17], intToBytes(reqLen, 4))

	// fmt.Printf("request msg in hex: %x\n", msg)
	// fmt.Printf("request msg: %v\n", msg)

	return msg
}

func BuildPieceMessage(pieceIdx, blockOffset int, block []byte) []byte {
	// <len=0009+X><id=7><index><begin><block>, where X denotes bytes needed for block
	// bytes: 4 + 1 + 4 + 4 + X

	msg := make([]byte, 13+len(block))
	var msgLen, msgID int = 9 + len(block), int(Piece)

	copy(msg[0:4], intToBytes(msgLen, 4))
	copy(msg[4:5], intToBytes(msgID, 1))
	copy(msg[5:9], intToBytes(pieceIdx, 4))
	copy(msg[9:13], intToBytes(blockOffset, 4))
	copy(msg[13:], block)

	fmt.Printf("piece msg in hex: %x", msg)

	return msg
}

// BuildCancelMessage is similar to BuildRequestMessage but varies in message ID
func BuildCancelMessage(pieceIdx, blockOffset, reqLen int) []byte {
	// <len=0013><id=8><index><begin><length>
	// bytes: 4 + 1 + 4 + 4 + 4 = 17
	msg := make([]byte, 17)
	var msgLen, msgID int = 13, int(Cancel)

	copy(msg[0:4], intToBytes(msgLen, 4))
	copy(msg[4:5], intToBytes(msgID, 1))
	copy(msg[5:9], intToBytes(pieceIdx, 4))
	copy(msg[9:13], intToBytes(blockOffset, 4))
	copy(msg[13:17], intToBytes(reqLen, 4))

	fmt.Printf("cancel msg in hex: %x", msg)

	return msg
}

func BuildPortMessage(port int) []byte {
	// <len=0003><id=9><listen-port>
	// bytes: 4 + 1 + 2
	msg := make([]byte, 7)
	var msgLen, msgID int = 13, int(Port)

	copy(msg[0:4], intToBytes(msgLen, 4))
	copy(msg[4:5], intToBytes(msgID, 1))
	copy(msg[5:7], intToBytes(port, 2))

	fmt.Printf("port msg in hex: %x", msg)

	return msg
}

// intToBytes converts num to number of bytes defined by byteCount
func intToBytes(num int, byteCount int) []byte {
	buf := new(bytes.Buffer)
	var err error

	switch byteCount {
	case 1:
		// use uint8
		err = binary.Write(buf, binary.BigEndian, uint8(num))
	case 2:
		// use uint32
		err = binary.Write(buf, binary.BigEndian, uint16(num))
	case 4:
		// use uint32
		err = binary.Write(buf, binary.BigEndian, uint32(num))
	default:
		return nil
	}

	if err != nil {
		log.Printf("error converting int to bytes: %d, %v", num, err)
	}

	return buf.Bytes()
}

func ParseMessage(msg []byte) (*Message, error) {
	if len(msg) == 0 {
		return nil, fmt.Errorf("message with no payload, possibly keep-alive")
	}

	// payload[0] denotes message ID
	msgID := messageID(msg[0])
	if !isValidMessageID(msgID) {
		return nil, fmt.Errorf("invalid message payload, message ID not valid, got %d", msgID)
	}

	return &Message{
		ID:      msgID,
		Payload: msg[1:],
	}, nil
}

func isValidMessageID(id messageID) bool {
	validIDs := []messageID{
		Choke,
		Unchoke,
		Interested,
		NotInterested,
		Have,
		Bitfield,
		Request,
		Piece,
		Cancel,
		Port,
	}

	for _, vid := range validIDs {
		if id == vid {
			return true
		}
	}

	return false
}
