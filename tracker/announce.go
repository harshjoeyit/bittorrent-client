package tracker

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"my-bittorrent/peer"
	"net"
)

type announceRequest struct {
	ConnectionID  int64    // Has to be the same obtained in connectResponse
	Action        int32    // default 1
	TransactionID int32    // randomly generated
	InfoHash      [20]byte // calculated based on torrent file
	PeerID        [20]byte
	Downloaded    int64 // bytes
	Left          int64 // bytes
	Uploaded      int64 // bytes
	Event         int32 // 0: none; 1: completed; 2: started; 3: stopped
	IPAddress     int32 // default 0
	Key           int32
	NumWant       int32
	Port          int16
}

func (req *announceRequest) toBytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, req)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize announce request: %v", err)
	}
	return buf.Bytes(), nil
}

func buildAnnounceRequest(connectionID int64, infoHash, peerID [20]byte, downloaded, left, uploaded int64, port int16) *announceRequest {
	req := &announceRequest{
		ConnectionID:  connectionID,
		Action:        1, // Announce
		TransactionID: randInt32(),
		InfoHash:      infoHash,
		PeerID:        peerID,
		Downloaded:    downloaded,
		Left:          left,
		Uploaded:      uploaded,
		Event:         0, // None (0)
		IPAddress:     0, // Default (0)
		Key:           randInt32(),
		NumWant:       -1, // Default (-1)
		Port:          port,
	}

	log.Printf("Announce request: %+v\n", req)

	return req
}

type IPv4AnnounceResponse struct {
	Action        int32
	TransactionID int32
	Interval      int32 // time interval before which announce request should not be re-triggered
	Leechers      int32
	Seeders       int32
	Peers         []*peer.Peer
}

func parseAnnounceResponse(data []byte) (*IPv4AnnounceResponse, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("invalid response: too short, expected > 20 bytes, got: %d", len(data))
	}

	resp := &IPv4AnnounceResponse{}
	buf := bytes.NewReader(data)

	// Parse fixed-size fields
	err := binary.Read(buf, binary.BigEndian, &resp.Action)
	if err != nil {
		return nil, fmt.Errorf("failed to parse action: %v", err)
	}

	err = binary.Read(buf, binary.BigEndian, &resp.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction ID: %v", err)
	}

	err = binary.Read(buf, binary.BigEndian, &resp.Interval)
	if err != nil {
		return nil, fmt.Errorf("failed to parse interval: %v", err)
	}

	err = binary.Read(buf, binary.BigEndian, &resp.Leechers)
	if err != nil {
		return nil, fmt.Errorf("failed to parse leechers: %v", err)
	}

	err = binary.Read(buf, binary.BigEndian, &resp.Seeders)
	if err != nil {
		return nil, fmt.Errorf("failed to parse seeders: %v", err)
	}

	// Parse Peers
	peerData := data[20:]
	for len(peerData) >= 6 {
		// Extract ip and port
		ip := net.IP(peerData[:4])
		port := binary.BigEndian.Uint16(peerData[4:6])

		// Append to the list of peers
		resp.Peers = append(resp.Peers, &peer.Peer{
			IPAddress: ip,
			Port:      port,
		})

		// Slice peerData to process next peers
		peerData = peerData[6:]
	}

	log.Printf("Announce response: %+v\n", resp)

	return resp, nil
}
