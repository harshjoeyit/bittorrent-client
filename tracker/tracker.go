package tracker

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"my-bittorrent/peer"
	"my-bittorrent/torrent"
	"net"
	"time"
)

const AnnounceReqPort int16 = 6881

type TrackerResponse int

const (
	Connect  TrackerResponse = 0
	Announce TrackerResponse = 1
)

func GetPeers(t *torrent.Torrent) ([]*peer.Peer, error) {
	announceUrl, err := t.GetAnnounceUrl()
	if err != nil {
		log.Printf("Error getting announce url: %v", err)
	}

	// hardcoded since primary announceUrl for torrent used for testing is not
	// returning any response
	announceUrl = "udp://tracker.opentrackr.org:1337"

	log.Printf("announceUrl: %s\n", announceUrl)

	// UDP conn
	conn, err := connectUDP(announceUrl)
	if err != nil {
		return nil, fmt.Errorf("error connecting to server address: %v", err)
	}
	defer conn.Close()

	// Send connection request
	req := buildConnectRequest()
	connReqBytes, err := req.toBytes()
	if err != nil {
		return nil, err
	}

	err = sendMessage(conn, connReqBytes)
	if err != nil {
		log.Printf("Error sending message: %v\n", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channels for results and errors
	peersCh := make(chan []*peer.Peer)
	errCh := make(chan error)

	go func() {
		defer close(peersCh)
		defer close(errCh)

		err := receiveMessage(ctx, conn, req, t, peersCh)
		if err != nil {
			errCh <- err
		}
	}()

	for {
		select {
		case peers := <-peersCh:
			// Successfully receveid peers
			return peers, nil
		case err := <-errCh:
			// Error received, stop processing
			return nil, err
		case <-ctx.Done():
			// Context cancellation
			return nil, ctx.Err()
		}
	}
}

// sendMessage sends message on the given udp connection
func sendMessage(conn *net.UDPConn, message []byte) error {
	_, err := conn.Write(message)
	if err != nil {
		return fmt.Errorf("error sending message: %v", err)
	}

	// log.Printf("Message sent: %x\n", message)
	log.Printf("Message sent")

	return nil
}

// receiveMessage listens on the udp connection for incoming messages
func receiveMessage(
	ctx context.Context,
	conn *net.UDPConn,
	connReq *connectRequest,
	t *torrent.Torrent,
	peersCh chan<- []*peer.Peer) error {
	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping receiveMessage due to context cancellation")
			return nil
		default:
			// receive response from tracker
			log.Printf("waiting for message...")

			bufferSize := 1024 // can set to 65535 (Max UDP payload size)
			buffer := make([]byte, bufferSize)

			conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			n, addr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				return fmt.Errorf("error reading response %v", err)
			}

			if n >= bufferSize {
				log.Printf("Warning: data might have been truncated\n")
			}

			resp := buffer[:n]
			log.Printf("Received response from %s, response: %v \n", addr, resp)

			responseType, err := getResponseType(resp)
			if err != nil {
				return fmt.Errorf("error reading response type: %v", err)
			}

			log.Printf("Response type: %d\n", responseType)

			switch responseType {
			case Connect:
				connResp, err := parseConnectResponse(resp)
				if err != nil {
					return fmt.Errorf("error parsing connect response: %v", err)
				}

				// validate response

				if connResp.TransactionID != connReq.TransactionID {
					return fmt.Errorf("error Transaction ID mismatch: expected: %d, got %d",
						connResp.TransactionID, connReq.TransactionID)
				}

				if connResp.Action != connReq.Action {
					return fmt.Errorf("error Action mismatch: expected: %d, got %d",
						connResp.Action, connReq.Action)
				}

				log.Printf("Successfully received connection ID: %d", connResp.ConnectionID)

				// build announce request

				// get info hash of torrent
				infoHash, err := t.GetInfoHash()
				if err != nil {
					return fmt.Errorf("error getting info hash: %v", err)
				}

				// get peer ID
				peerID, err := peer.GetPeerID()
				if err != nil {
					return fmt.Errorf("error getting peer ID: %v", err)
				}

				fileLength, err := t.GetFileLength()
				if err != nil {
					return fmt.Errorf("error getting file size: %v", err)
				}

				// send announce request
				announceReq := buildAnnounceRequest(connResp.ConnectionID, infoHash, peerID, 0, fileLength, 0, AnnounceReqPort)
				announceReqBytes, err := announceReq.toBytes()
				if err != nil {
					return fmt.Errorf("error converting announce request to bytes: %v", err)
				}

				sendMessage(conn, announceReqBytes)

			case Announce:
				announceRes, err := parseAnnounceResponse(resp)
				if err != nil {
					return fmt.Errorf("error parsing announce response: %v", err)
				}

				peersCh <- announceRes.Peers

				return nil // we don't need to wait for any more messages

			default:
				return fmt.Errorf("error invalid response type: %s", resp)
			}
		}
	}
}

// getResponseType returns the type of response received
// UDP tracker protocol definition - https://www.bittorrent.org/beps/bep_0015.html
func getResponseType(data []byte) (TrackerResponse, error) {
	// both connect and announce response have int32 at offset 0 which denote action
	// 0 refers to connect and 1 refers to announce
	var action int32
	buf := bytes.NewReader(data)

	err := binary.Read(buf, binary.BigEndian, &action)
	if err != nil {
		return -1, err
	}

	if action == 0 {
		return Connect, nil
	} else if action == 1 {
		return Announce, nil
	} else {
		return -1, nil
	}
}
