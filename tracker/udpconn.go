package tracker

import (
	"fmt"
	"log"
	"net"
	"net/url"
)

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
