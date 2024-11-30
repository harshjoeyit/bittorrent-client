package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"my-bittorrent/decoder"
	"my-bittorrent/peer"
	"my-bittorrent/torrent"
	"my-bittorrent/tracker"
)

func main() {
	relFilepath := os.Args[1]

	// Generate Peer ID
	_, err := peer.GetPeerID()
	if err != nil {
		log.Println("error: failed to generate peer ID")
		return
	}

	if relFilepath == "" {
		log.Println("error: filepath not supplied")
		return
	}

	bencoded, err := readFile(relFilepath)
	if err != nil {
		log.Printf("Error reading torrent file: %v", err)
		return
	}

	decoded, err := decoder.DecodeBencode(bencoded)
	if err != nil {
		log.Printf("Error decoding bencode: %v", err)
		return
	}

	// create a new torrent instance
	t, err := torrent.NewTorrent(decoded)
	if err != nil {
		log.Printf("Error creating New Torrent: %v", err)
		return
	}

	// get peers
	peers, err := tracker.GetPeers(t)
	if err != nil {
		log.Printf("Error in getting peers: %v", err)
		return
	}
	// peers := peer.GetCachedPeers()
	// _ = peers

	log.Printf("successfully received peers\n")
	for i, p := range peers {
		fmt.Printf("i: %d, IP: %s, port: %d\n", i, p.IPAddress, p.Port)
	}

	// Peers which our client is successfully connected to
	connectedPeers := Connect(peers)

	handshakeMsg, err := peer.BuildHandshakeMessage(t.InfoHash)
	if err != nil {
		log.Printf("error building handshake msg: %v", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(connectedPeers))

	// Send handshake and start receiving messages
	for _, p := range connectedPeers {
		ctx, cancel := context.WithCancel(context.Background())

		go func(p *peer.Peer) {
			defer wg.Done()

			fmt.Println("START: to receive messages...", p.IPAddress, ":", p.Port)
			peer.ReceiveMessages(ctx, p, t)
			fmt.Println("END: receive messages", p.IPAddress, ":", p.Port)
		}(p)

		fmt.Println("START: sending handshake message", p.IPAddress, ":", p.Port)
		sendErr := peer.SendMessage(p.Conn, handshakeMsg)
		if sendErr != nil {
			fmt.Printf("Error in sending handshake msg to %s:%d: %v\n", p.IPAddress, p.Port, sendErr)
			cancel()
		}
		fmt.Println("END: sending handshake", p.IPAddress, ":", p.Port)
	}

	wg.Wait()
}

func readFile(relFilepath string) ([]byte, error) {
	// read complete file into memory at once
	data, err := os.ReadFile(relFilepath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Connect establishes connection with each peer
func Connect(peers []*peer.Peer) []*peer.Peer {
	isConnected := make([]bool, len(peers))
	retryTimes := 2
	delay := 5 * time.Second

	var wg sync.WaitGroup

	remaining := len(peers) // peers which are yet to be connected
	for i := 0; i < retryTimes; i++ {
		if i > 0 {
			// delay before retrying
			fmt.Println("waiting before retry")
			time.Sleep(delay)
		}

		wg.Add(remaining)
		go tryConnecting(peers, &wg, isConnected)
		wg.Wait()

		connected := countConnected(isConnected)
		remaining = remaining - connected
		fmt.Println("connected: ", connected)
	}

	var connectedPeers []*peer.Peer
	for i, peer := range peers {
		if isConnected[i] {
			connectedPeers = append(connectedPeers, peer)
		}
	}

	return connectedPeers
}

func tryConnecting(peers []*peer.Peer, wg *sync.WaitGroup, isConnected []bool) {
	for i, p := range peers {
		if isConnected[i] {
			continue
		}

		go func() {
			defer wg.Done()
			conn, err := peer.ConnectTCP(p)
			if err != nil {
				log.Println(err)
				return
			}

			log.Printf("Connected to peer %s", conn.RemoteAddr().String())
			isConnected[i] = true
			p.Conn = conn // update connection
		}()
	}
}

func countConnected(isConnected []bool) int {
	var c int
	for _, val := range isConnected {
		if val {
			c++
		}
	}
	return c
}

func Download(t *torrent.Torrent, peer *peer.Peer) {

}
