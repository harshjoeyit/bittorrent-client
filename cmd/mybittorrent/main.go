package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"

	"my-bittorrent/decoder"
)

func readTorrentFile(relFilepath string) (string, error) {
	// open the file
	file, err := os.Open(relFilepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	content, err := reader.ReadBytes(0) // Read until EOF
	if err != nil {
		if err.Error() != "EOF" { // EOF is expected at the end
			return "", err
		}
	}

	// Print the file content as a UTF-8 string
	// fmt.Println(string(content))
	return string(content), nil
}

func getAnnounceUrl(decoded interface{}) (string, error) {
	// type assert to map[string]interface{}
	decodedMap, ok := decoded.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("decoded data is not a map")
	}

	// check if announce field exists
	announce, ok := decodedMap["announce"]
	if !ok {
		return "", fmt.Errorf("announce field does not exist")
	}

	// type assert to string
	announceUrl, ok := announce.(string)
	if !ok {
		return "", fmt.Errorf("announce field does not exist")
	}

	return announceUrl, nil
}

func sendMessage(announceUrl string) {
	parsedUrl, err := url.Parse(announceUrl)
	if err != nil {
		log.Printf("Error in parsing url %v\n", err)
	}

	host := parsedUrl.Host
	if host == "" {
		log.Printf("Invalid URL, host missing\n")
		return
	}

	log.Printf("Tracker Host: %s\n", host)

	trackerServerAddr, err := net.ResolveUDPAddr("udp4", host)
	if err != nil {
		log.Printf("Error resolving server address: %v\n", err)
		return
	}

	log.Printf("Resolved address: %v\n", trackerServerAddr)

	// connect to tracker client
	conn, err := net.DialUDP("udp4", nil, trackerServerAddr)
	if err != nil {
		log.Printf("Error connecting to server address: %v\n", err)
		return
	}
	defer conn.Close()

	// send message
	message := "Hello, Tracker!"
	_, err = conn.Write([]byte(message))
	if err != nil {
		log.Printf("Error sending message: %v\n", err)
		return
	}
	log.Printf("Message sent: %s\n", message)

	// receive response from tracker
	buffer := make([]byte, 1024)
	n, addr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		log.Printf("Error reading response: %v\n", err)
		return
	}
	log.Printf("Received response: %s from %s\n", string(buffer[:n]), addr)
}

func main() {
	command := os.Args[1]

	if command == "open" {
		relFilepath := os.Args[2]
		if relFilepath == "" {
			log.Println("error: filepath not supplied")
			return
		}

		bencodedStr, err := readTorrentFile(relFilepath)
		if err != nil {
			log.Printf("Error reading torrent file: %v", err)
			return
		}

		decoded, err := decoder.DecodeBencode(bencodedStr)
		if err != nil {
			log.Printf("Error decoding bencode: %v", err)
			return
		}

		announceUrl, err := getAnnounceUrl(decoded)
		if err != nil {
			log.Printf("Error getting announce url: %v", err)
		}

		// send message
		sendMessage(announceUrl)

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
