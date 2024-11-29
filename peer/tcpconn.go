package peer

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"my-bittorrent/torrent"
	"net"
	"time"
	"unicode/utf8"
)

func ConnectTCP(peer *Peer) (net.Conn, error) {
	// Create the address string in the format "IP:Port"
	addr := fmt.Sprintf("%s:%d", peer.IPAddress.String(), peer.Port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer %s: %w", addr, err)
	}

	// // Keep alive
	// tcpConn, ok := conn.(*net.TCPConn)
	// if ok {
	// 	err := tcpConn.SetKeepAlive(true)
	// 	if err != nil {
	// 		log.Println("failed to enable keep-alive: %v\n", err)
	// 	}
	// }

	return conn, nil
}

func SendMessage(conn net.Conn, message []byte) error {
	_, err := conn.Write(message)
	if err != nil {
		return fmt.Errorf("failed to send message to peer %s: %w", conn.RemoteAddr().String(), err)
	}

	return nil
}

func ReceiveMessages(ctx context.Context, peer *Peer, t *torrent.Torrent) {
	defer peer.Conn.Close()
	isHandshake := true // first message is handshake message

	for {
		select {
		// When we don't need connection anymore, we can simply stop reading and close connection
		case <-ctx.Done():
			log.Printf("stopping read loop for connection to %s as ctx cancelled or timeout\n", peer.Conn.RemoteAddr().String())
			return
		default:
			if isHandshake {
				msg, err := ReadHandshakeMessage(ctx, peer.Conn)
				if err != nil {
					log.Printf("error reading message: %v\n", err)
					return
				}

				fmt.Printf("handshake message received: %v\n", msg)

				// validate message
				if valErr := IsHandshakeMessageValid(msg, t.InfoHash); valErr != nil {
					// stop receiving messages
					log.Println(valErr)
					return
				}

				// copy(peer.ID, msg[48:68])

				// Handshake received and validated, now we are ready to
				// receive other messages
				isHandshake = false
				continue
			}

			// Read messages from the connection
			msg, err := ReadMessage(ctx, peer.Conn)
			if err != nil {
				log.Printf("error reading message: %v\n", err)
				return
			}

			// Print message
			fmt.Printf("message received: %v\n", msg)

			// Parse message
			parsedMsg, err := ParseMessage(msg)
			if err != nil {
				log.Printf("error parsing message: %v\n", err)
				continue
			}

			// route the message to handlers
			messageRouter(parsedMsg)
		}
	}
}

func ReadHandshakeMessage(ctx context.Context, conn net.Conn) ([]byte, error) {
	// Set a deadline
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetReadDeadline(time.Time{})

	// Decode message length
	msgLen := 49 + utf8.RuneCountInString(ProtocolIdentifier)

	// Channel for reading the message body
	done := make(chan error, 1)

	// Read message
	var msg []byte
	go func() {
		msg = make([]byte, msgLen)
		_, err := io.ReadFull(conn, msg)
		done <- err
	}()

	// Wait for read to complete or context done
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("failed to read handshake message as ctx cancelled or timeout %w", ctx.Err())
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("error reading handshake message %w", readError(err))
		}
	}

	return msg, nil
}

func IsHandshakeMessageValid(responseMsg []byte, expectedInfoHash [20]byte) error {
	handshakeMsgLen := 49 + utf8.RuneCountInString(ProtocolIdentifier)

	if len(responseMsg) < handshakeMsgLen {
		return fmt.Errorf("invalid handshake: response too short, got: %d bytes only", handshakeMsgLen)
	}

	// Validate pstrlen
	pstrlen := int(responseMsg[0])
	if pstrlen != len(ProtocolIdentifier) {
		return fmt.Errorf("invalid handshake: pstrlen mismatch, got: %d, expected: %d", pstrlen, len(ProtocolIdentifier))
	}

	// Validate pstr
	pstr := string(responseMsg[1 : 1+pstrlen])
	if pstr != ProtocolIdentifier {
		return fmt.Errorf("invalid handshake: pstr mismatch, expected '%s' got '%s'", ProtocolIdentifier, pstr)
	}

	// Validate info hash
	infoHashStart := 1 + pstrlen + 8
	infoHash := responseMsg[infoHashStart : infoHashStart+20]
	if !bytes.Equal(infoHash, expectedInfoHash[:]) {
		return fmt.Errorf("invalid handshake: info_hash mismatch")
	}

	// If tracker returns the peer ID,
	// verify Peer ID with what is returned from the tracker,

	return nil
}

// ReadMessage reads length prefix message from connection to peers
// such as choke, unchoke, request, piece, etc
func ReadMessage(ctx context.Context, conn net.Conn) ([]byte, error) {
	// Set a deadline
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetReadDeadline(time.Time{})

	const headerLen = 4 // 4 bytes for the length prefix

	// Channel to signal completion or error
	done := make(chan error, 1)

	// Read header
	var header []byte
	go func() {
		header = make([]byte, headerLen)
		_, err := io.ReadFull(conn, header)
		done <- err
	}()

	// Wait for read to complete or context done
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("failed to read message header as ctx cancelled or timeout %v", ctx.Err())
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("error reading message header %v", readError(err))
		}
	}

	// Decode message length
	msgLen := binary.BigEndian.Uint32(header)

	// Create another channel for reading the message body
	done = make(chan error, 1)

	// Read message
	var msg []byte
	go func() {
		msg = make([]byte, msgLen)
		_, err := io.ReadFull(conn, msg)
		done <- err
	}()

	// Wait for read to complete or context done
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("failed to read message as ctx cancelled or timeout %w", ctx.Err())
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("error reading message %w", readError(err))
		}
	}

	return msg, nil
}

// getReadErrorType returns the formatted the error message which is
// more verbose and clear based for the error
func readError(err error) error {
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("connection closed by peer: %w", err)
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return fmt.Errorf("read timeout: %w", err)
	}
	return fmt.Errorf("unexpected error: failed to read message: %w", err)
}

// messageRouter routes received message from peers to
// respective message handlers
func messageRouter(m *Message) {
	switch m.ID {
	case 0:
		chokeMsgHandler()
	case 1:
		unchokeMsgHandler()
	case 4:
		haveMsgHandler(m.Payload)
	case 5:
		bitfieldMsgHandler(m.Payload)
	case 7:
		pieceMsgHandler(m.Payload)
	}
}

// func SendMessage(conn net.Conn, msgID messageID, payload interface{}) error {
// 	switch msgID {
// 	case Choke:
// 		return sendChokeMessage(conn)
// 	default:
// 		return fmt.Errorf("invalid messageID")
// 	}
// }

// // isConnectionClosed checks if the connection is closed without consuming data
// func isConnectionClosed(conn net.Conn) bool {
// 	// using bufio.Reader so that we can peek
// 	reader := bufio.NewReader(conn)

// 	// Set a short read deadline to avoid indefinite blocking
// 	conn.SetReadDeadline(time.Now().Add(1 * time.Second))

// 	// Peek at one byte without consuming it
// 	_, err := reader.Peek(1)
// 	if err != nil {
// 		if err == io.EOF {
// 			fmt.Printf("Connection closed by peer %s: %v\n", conn.RemoteAddr().String(), err)
// 			return true
// 		}

// 		netErr, ok := err.(net.Error)
// 		if ok && netErr.Timeout() {
// 			fmt.Printf("Connection is still open but idle %s: %v\n", conn.RemoteAddr().String(), err)
// 			return false
// 		}

// 		fmt.Printf("Unexpected error: %s: %v\n", conn.RemoteAddr().String(), err)
// 		return true
// 	}

// 	fmt.Println("Connection is open.")
// 	return false
// }
