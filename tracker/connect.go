package tracker

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

// connectRequest defines format of connect request to tracker
// UDP tracker protocol definition - https://www.bittorrent.org/beps/bep_0015.html
type connectRequest struct {
	ProtocolID    int64 // hardoded magic number - 0x41727101980
	Action        int32 // Action is '0' for connectRequest
	TransactionID int32 // randomly generated
}

func buildConnectRequest() *connectRequest {
	req := &connectRequest{
		ProtocolID:    0x41727101980, // magic constant
		Action:        0,             // Default (0)
		TransactionID: randInt32(),
	}

	log.Printf("Connect request: %+v", req)

	return req
}

func (req *connectRequest) toBytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write fields in network byte order (big-endian)
	err := binary.Write(buf, binary.BigEndian, req)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize connect request: %v", err)
	}

	// log.Printf("ToBytes in hex: %x", buf.Bytes())

	return buf.Bytes(), nil
}

// connectResponse defines format of response to connect request to tracker
// UDP tracker protocol definition - https://www.bittorrent.org/beps/bep_0015.html
type connectResponse struct {
	Action        int32
	TransactionID int32
	ConnectionID  int64
}

// parseConnectResponse parses bytes to type connectResponse
func parseConnectResponse(data []byte) (*connectResponse, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("data too short: expected 16 bytes, got %d", len(data))
	}

	resp := &connectResponse{}
	buf := bytes.NewReader(data)

	err := binary.Read(buf, binary.BigEndian, &resp.Action)
	if err != nil {
		return nil, fmt.Errorf("error in reading 'Action' in connect response: %v", err)
	}

	err = binary.Read(buf, binary.BigEndian, &resp.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("error in reading 'TransactionID' in connect response: %v", err)
	}

	err = binary.Read(buf, binary.BigEndian, &resp.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("error in reading 'ConnectionID' in connect response: %v", err)
	}

	log.Printf("Connect response: %+v\n", resp)

	return resp, nil
}
