package tracker

import (
	"bytes"
	"context"
	cryptoRand "crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/url"
	"time"

	"github.com/jackpal/bencode-go"
)

const PORT_ANNOUNCE_REQ int16 = 6881

type TrackerResponse int

const CONNECT TrackerResponse = 0
const ANNOUNCE TrackerResponse = 1

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

func (req *connectRequest) toBytes() []byte {
	buf := new(bytes.Buffer)
	// Write fields in network byte order (big-endian)
	binary.Write(buf, binary.BigEndian, req.ProtocolID)
	binary.Write(buf, binary.BigEndian, req.Action)
	binary.Write(buf, binary.BigEndian, req.TransactionID)

	// log.Printf("ToBytes in hex: %x", buf.Bytes())

	return buf.Bytes()
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

func randInt32() int32 {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	return r.Int31()
}

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
	Peers         []Peer
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
		resp.Peers = append(resp.Peers, Peer{
			IPAddress: ip,
			Port:      port,
		})

		// Slice peerData to process next peers
		peerData = peerData[6:]
	}

	log.Printf("Announce response: %+v\n", resp)

	return resp, nil
}

var peerID [20]byte

func getPeerID() ([20]byte, error) {
	// Check if the peerID has already been generated
	if peerID != [20]byte{} {
		return peerID, nil
	}

	// Client identifier (e.g., "AT" for Azureus-style)
	//
	// Azureus-style uses the following encoding: '-', two characters for
	// client id, four ascii digits for version number, '-', followed by
	// random numbers. For example: '-AT0001-'...
	clientID := "-AT0001-" // 8 bytes

	copy(peerID[:], clientID)

	// Generate the remaining part randomly
	randomBytes := make([]byte, 12)
	_, err := cryptoRand.Read(randomBytes)
	if err != nil {
		return peerID, fmt.Errorf("error in generating random bytes %v", err)
	}

	copy(peerID[len(clientID):], randomBytes)

	return peerID, nil
}

// Peer represents a single node participating in a torrent network
type Peer struct {
	IPAddress net.IP
	Port      uint16
}

func GetPeers(torrent interface{}) ([]Peer, error) {
	announceUrl, err := getAnnounceUrl(torrent)
	if err != nil {
		log.Printf("Error getting announce url: %v", err)
	}

	// hardcoded since primary announceUrl for torrent used for testing is not
	// returning any response
	announceUrl = "udp://tracker.opentrackr.org:1337"

	fmt.Printf("announceUrl: %s\n", announceUrl)

	conn, err := getUDPConn(announceUrl)
	if err != nil {
		return nil, fmt.Errorf("error connecting to server address: %v", err)
	}
	defer conn.Close()

	// connection request
	req := buildConnectRequest()

	// send connection request
	err = sendMessage(conn, req.toBytes())
	if err != nil {
		log.Printf("Error sending message: %v\n", err)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// channel to receive peer list
	ch := make(chan []Peer)

	go func() {
		receiveMessage(ctx, conn, req, torrent, ch)
	}()

	peers := <-ch
	for i, peer := range peers {
		fmt.Printf("peer i: %d, peer: %+v\n", i, peer)
	}

	// stop receiving messages
	cancel()

	return peers, nil
}

// getAnnounceUrl extracts and returns annouce url (tracker url)
// from torrent
func getAnnounceUrl(torrent interface{}) (string, error) {
	// type assert to map[string]interface{}
	torrentMap, ok := torrent.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("decoded data is not a map")
	}

	// check if announce field exists
	announce, ok := torrentMap["announce"]
	if !ok {
		return "", fmt.Errorf("'announce' field does not exist")
	}

	// type assert to string
	announceUrl, ok := announce.(string)
	if !ok {
		return "", fmt.Errorf("'announce' field is not a string")
	}

	return announceUrl, nil
}

func getInfoHash(torrent interface{}) ([20]byte, error) {
	infoHash := [20]byte{0}

	// type assert to map[string]interface{}
	torrentMap, ok := torrent.(map[string]interface{})
	if !ok {
		return infoHash, fmt.Errorf("decoded data is not a map")
	}

	// check if info field exists
	info, ok := torrentMap["info"]
	if !ok {
		return infoHash, fmt.Errorf("'info' field does not exist")
	}

	// type assert
	infoMap, ok := info.(map[string]interface{})
	if !ok {
		return infoHash, fmt.Errorf("'info' field not a map")
	}

	// buffer to store bencoded 'info'
	var buf bytes.Buffer

	err := bencode.Marshal(&buf, infoMap)
	if err != nil {
		return infoHash, fmt.Errorf("info could not be bencoded")
	}

	infoHash = sha1.Sum(buf.Bytes())

	// prints 40 character hexadecimal string
	log.Printf("info_hash: %s\n", hex.EncodeToString(infoHash[:]))

	// converting to string - not required
	// infoHashStr = hex.EncodeToString(infoHash[:])

	return infoHash, nil
}

// returns file size (in bytes)
func getFileSize(torrent interface{}) (int64, error) {
	var fileSize int64

	// type assert to map[string]interface{}
	torrentMap, ok := torrent.(map[string]interface{})
	if !ok {
		return fileSize, fmt.Errorf("decoded data is not a map")
	}

	// check if info field exists
	info, ok := torrentMap["info"]
	if !ok {
		return fileSize, fmt.Errorf("'info' field does not exist")
	}

	// type assert
	infoMap, ok := info.(map[string]interface{})
	if !ok {
		return fileSize, fmt.Errorf("'info' field not a map")
	}

	/*
		For a single-file torrent: length (value in the info dictionary) gives
		the total size.

		{
			"announce": "http://tracker.example.com:8080/announce",
			"info": {
				"length": 1024000
			},
		}
	*/

	// check if 'length' key in present in dict, if yes, it's a single file

	length, ok := infoMap["length"]
	if ok {
		fileSize, ok := length.(int64)
		if !ok {
			return fileSize, fmt.Errorf("info.length field is not int64")
		}

		return fileSize, nil
	}

	// !ok - multiple files

	/*
		For a multi-file torrent: Add up the sizes of all files listed in the
		files array under the info dictionary.

		{
			"announce": "http://tracker.example.com:8080/announce",
			"info": {
				"name": "example_folder",
				"piece length": 262144,
				"pieces": "abcd1234efgh5678...",
				"files": [
					{
						"length": 512000,
						"path": ["file1.txt"]
					},
					{
						"length": 1024000,
						"path": ["subfolder2", "file2.txt"]
					}
				]
			}
		}
	*/

	files, ok := infoMap["files"]
	if !ok {
		return fileSize, fmt.Errorf("info.files field is not exist")
	}

	// type asset - files should be a slice (list)
	fileList, ok := files.([]interface{})
	if !ok {
		return fileSize, fmt.Errorf("info.files field is not a list")
	}

	for i, file := range fileList {
		// each file should be a map[string]{}
		fileMap, ok := file.(map[string]interface{})
		if !ok {
			return fileSize, fmt.Errorf("info.files.file value is not of type map[string]interface{}")
		}

		// check if length field exists
		length, ok := fileMap["length"]
		if ok {
			size, ok := length.(int64)
			if !ok {
				return fileSize, fmt.Errorf("info.files.file[%d].length field is not int64", i)
			}

			// add to the total size
			fileSize += size
		}
	}

	return fileSize, nil
}

// getUDPConn returns a connection for the given url
func getUDPConn(serverUrl string) (*net.UDPConn, error) {
	// parse url to get host
	parsedUrl, err := url.Parse(serverUrl)
	if err != nil {
		log.Printf("Error in parsing url %v\n", err)
	}

	host := parsedUrl.Host
	if host == "" {
		return nil, fmt.Errorf("invalid URL, host missing")
	}

	log.Printf("Tracker Host: %s\n", host)

	// resolve host
	serverAddr, err := net.ResolveUDPAddr("udp4", host)
	if err != nil {
		return nil, fmt.Errorf("error resolving server address: %v", err)
	}

	log.Printf("Resolved address: %v\n", serverAddr)

	// connect to tracker client
	return net.DialUDP("udp4", nil, serverAddr)
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
func receiveMessage(ctx context.Context, conn *net.UDPConn, connReq *connectRequest, torrent interface{}, ch chan<- []Peer) {
	for {
		fmt.Println("**** receiveMessage: inside for loop")

		select {
		case <-ctx.Done():
			log.Println("Stopping receiveMessage due to context cancellation")
			return
		default:
			// receive response from tracker
			log.Printf("waiting for message...")

			bufferSize := 1024 // can set to 65535 (Max UDP payload size)
			buffer := make([]byte, bufferSize)

			conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			n, addr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				log.Printf("Error reading response: %v\n", err)
				return // or retry
			}
			if n >= bufferSize {
				log.Printf("Warning: data might have been truncated\n")
			}

			resp := buffer[:n]

			log.Printf("Received response from %s, response: %v \n", addr, resp)

			responseType, err := getResponseType(resp)
			if err != nil {
				log.Printf("Error reading response type: %v\n", err)
				return // or retry
			}

			log.Printf("Response type: %d\n", responseType)

			switch responseType {
			case CONNECT:
				connResp, err := parseConnectResponse(resp)
				if err != nil {
					log.Printf("error parsing connect response: %v\n", err)
					continue
				}

				// validate response
				if connResp.TransactionID != connReq.TransactionID {
					log.Printf("Transaction ID mismatch: expected: %d, got %d",
						connResp.TransactionID, connReq.TransactionID)
					continue
				}

				if connResp.Action != connReq.Action {
					log.Printf("Action mismatch: expected: %d, got %d",
						connResp.Action, connReq.Action)
					continue
				}

				log.Printf("Successfully received connection ID: %d", connResp.ConnectionID)

				// build announce request

				// get info hash of torrent
				infoHash, err := getInfoHash(torrent)
				if err != nil {
					log.Printf("Erorr getting info hash: %v", err)
					continue
				}

				// get peer ID
				peerID, err := getPeerID()
				if err != nil {
					log.Printf("Erorr getting peer ID: %v", err)
					continue
				}

				fileSize, err := getFileSize(torrent)
				if err != nil {
					log.Printf("Erorr getting file size: %v", err)
					continue
				}

				// announce request
				announceReq := buildAnnounceRequest(connResp.ConnectionID, infoHash, peerID, 0, fileSize, 0, PORT_ANNOUNCE_REQ)
				// send announce request
				announceReqBytes, err := announceReq.toBytes()
				if err != nil {
					log.Printf("Erorr converting announce request to bytes: %v", err)
					continue
				}

				sendMessage(conn, announceReqBytes)

			case ANNOUNCE:
				announceRes, err := parseAnnounceResponse(resp)
				if err != nil {
					log.Printf("error parsing announce response: %v\n", err)
					return
				}

				ch <- announceRes.Peers

				return // we don't need to wait for any more messages

			default:
				log.Printf("Invalid response type: %s", resp)
				return
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
		return CONNECT, nil
	} else if action == 1 {
		return ANNOUNCE, nil
	} else {
		return -1, nil
	}
}
