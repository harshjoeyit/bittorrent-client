package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"my-bittorrent/decoder"
	"my-bittorrent/tracker"

	"github.com/jackpal/bencode-go"
	// "github.com/zeebo/bencode"
)

func readTorrentFile(relFilepath string) ([]byte, error) {
	// open the file
	data, err := os.ReadFile(relFilepath)
	if err != nil {
		return nil, err
	}

	// Print the file content as a UTF-8 string
	return data, nil
}

func main() {
	command := os.Args[1]

	if command == "open" {
		relFilepath := os.Args[2]
		if relFilepath == "" {
			log.Println("error: filepath not supplied")
			return
		}

		bencoded, err := readTorrentFile(relFilepath)
		if err != nil {
			log.Printf("Error reading torrent file: %v", err)
			return
		}

		// torrent, err := decoder.DecodeBencode(string(bencoded))
		torrent, err := DecodeUsingPackage(bencoded)
		if err != nil {
			log.Printf("Error decoding bencode: %v", err)
			return
		}

		// get peers
		peers, err := tracker.GetPeers(torrent)
		if err != nil {
			log.Printf("Error in getting peers: %v", err)
			return
		}

		log.Printf("peers:%v\n", peers)

	} else if command == "decode" {
		bencodedValue := os.Args[2]

		decoded, err := decoder.DecodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}

// temporary using package to parse the bencoded torrent
func DecodeUsingPackage(bencoded []byte) (interface{}, error) {
	var data interface{}
	var err error

	buf := bytes.NewReader(bencoded)
	data, err = bencode.Decode(buf)
	if err != nil {
		fmt.Println("error in decoding in main.go", err)
		return data, err
	}

	return data, err
}
