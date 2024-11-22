package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"my-bittorrent/decoder"
	"my-bittorrent/tracker"
)

func readFile(relFilepath string) ([]byte, error) {
	// read complete file into memory at once
	data, err := os.ReadFile(relFilepath)
	if err != nil {
		return nil, err
	}

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

		// get peers
		peers, err := tracker.GetPeers(decoded)
		if err != nil {
			log.Printf("Error in getting peers: %v", err)
			return
		}

		log.Printf("peers:%v\n", peers)

	} else if command == "decode" {
		relFilepath := os.Args[2]

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

		torrentJson, _ := json.Marshal(decoded)
		fmt.Println(len(string(torrentJson)))

	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}

// temporary using package to parse the bencoded torrent
// func DecodeUsingPackage(bencoded []byte) (interface{}, error) {
// 	var data interface{}
// 	var err error

// 	buf := bytes.NewReader(bencoded)
// 	data, err = bencode.Decode(buf)
// 	if err != nil {
// 		fmt.Println("error in decoding in main.go", err)
// 		return data, err
// 	}

// 	return data, err
// }
