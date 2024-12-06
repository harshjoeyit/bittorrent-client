package peer

import (
	"encoding/binary"
	"fmt"
	"log"
	"my-bittorrent/queue"
	"my-bittorrent/torrent"
)

func chokeMsgHandler(p *Peer) error {
	fmt.Printf("CHOKE message received from: %s\n", p.Conn.RemoteAddr().String())
	// close the connection
	// return p.Conn.Close()
	p.AmChoked = true
	return nil
}

func unchokeMsgHandler(p *Peer, t *torrent.Torrent) error {
	log.Printf("UNCHOKE message received from: %s\n", p.Conn.RemoteAddr().String())

	p.AmChoked = false

	// Request a piece upon unchoking
	// It is possible that TaskQueue is filled with blocks but the client
	// was choked before it could request for all the blocks,
	// hence when client gets unchoked again, request for blocks
	err := requestOnePiece(p, t.Downloader)
	if err != nil {
		return fmt.Errorf("error requesting one piece: %w", err)
	}

	return nil
}

func requestOnePiece(p *Peer, d *torrent.Downloader) error {
	if p.AmChoked {
		return fmt.Errorf("cannot request for piece as peer choking: %s", p.Conn.RemoteAddr().String())
	}

	// Among pieces that the peer has, find a piece which is needed
	// and request for it
	for !p.TaskQueue.IsEmpty() {
		var b *queue.Block = p.TaskQueue.Pop()

		if !d.IsNeeded(b) {
			continue
		}

		err := SendMessage(p.Conn, BuildRequestMessage(b.PieceIdx, b.BlockOffset, b.BlockLength))
		if err != nil {
			return fmt.Errorf("error sending message: %w", err)
		}

		d.Requested(b)

		fmt.Printf("requested [piece][block] [%d][%d] from: %s\n", b.PieceIdx, d.BlockIdx(b), p.Conn.RemoteAddr().String())

		// break since we are requesting for only one piece
		// Following strategy to request pieces from peer with highest upload rate,
		// rest of the pieces will be requested when response for this piece is received
		break
	}

	return nil
}

func haveMsgHandler(payload []byte, p *Peer, t *torrent.Torrent) error {
	fmt.Printf("HAVE message received from %s\n", p.Conn.RemoteAddr().String())

	if len(payload) != 4 {
		return fmt.Errorf("payload for have message should be 4 bytes, got %d", payload)
	}

	// payload contains the piece index
	pi := binary.BigEndian.Uint32(payload)
	pieceIdx := int(pi)

	fmt.Println("HAVE: ", pieceIdx)

	e := p.TaskQueue.IsEmpty()

	err := enqueueBlocksForPiece(pieceIdx, p, t)
	if err != nil {
		return fmt.Errorf("error enqueuing blocks for piece: %d, error: %w", pieceIdx, err)
	}

	// if Task queue was empty, request a piece to start receiving pieces
	if e {
		return requestOnePiece(p, t.Downloader)
	}

	return nil
}

func bitfieldMsgHandler(payload []byte, p *Peer, t *torrent.Torrent) error {
	log.Printf("BITFIELD message received, with len: %d, from peer: %s", len(payload), p.Conn.RemoteAddr().String())

	var pieceIndices []int

	// Process byte by byte
	for i, b := range payload {
		adder := i * 8
		// Check all the 8 bits
		for i := 0; i < 8; i++ {
			// check if bit is set
			if (1<<i)&b > 0 {
				// index in current byte = 8 - i, since MSB denotes lower index
				idx := 7 - i
				pieceIndices = append(pieceIndices, adder+idx)
			}
		}
	}

	// fmt.Println("BITFIELD decoded indices: ", pieceIndices)

	e := p.TaskQueue.IsEmpty()

	// Enqueue all the blocks for the pieces received in bifield
	for i := 0; i < len(pieceIndices); i++ {
		err := enqueueBlocksForPiece(i, p, t)
		if err != nil {
			return fmt.Errorf("error pushing blocks in queue: %w", err)
		}
	}

	// if Task queue was empty, request a piece to start receiving pieces
	if e {
		return requestOnePiece(p, t.Downloader)
	}

	return nil
}

// enqueueBlocksForPiece pushes all the blocks for a pieces in the queue for a peer
// this function is triggered when have and bitfield messages are received
func enqueueBlocksForPiece(pieceIdx int, p *Peer, t *torrent.Torrent) error {
	// Enqueue all the blocks for this piece
	blocks, err := t.GetBlocksCount(pieceIdx)
	if err != nil {
		return fmt.Errorf(
			"error pushing blocks in queue, failed to get blocks count for piece: %d error: %w",
			pieceIdx, err)
	}

	for i := 0; i < blocks; i++ {
		blockOffset := i * torrent.DefaultBlockLength
		blockLength, err := t.GetBlockLength(pieceIdx, i)
		if err != nil {
			return fmt.Errorf(
				"error pushing blocks in queue, failed to get block length for piece: %d, blocks %d error: %w",
				pieceIdx, i, err)
		}

		p.TaskQueue.Push(
			queue.NewBlock(
				int(pieceIdx),
				blockOffset,
				blockLength,
			))
	}

	// fmt.Printf("Enqueued all pieces for pieceIdx: %d, for peer: %s\n", pieceIdx, p.Conn.RemoteAddr().String())

	return nil
}

func pieceMsgHandler(payload []byte, p *Peer, t *torrent.Torrent) error {
	// log.Printf("PIECE message received, with len: %d, from peer: %s\n", len(payload), p.Conn.RemoteAddr().String())

	b, blockData := parsePieceMsg(payload)

	// Validate block

	// Validate indices (if such a block exists)
	if !t.Downloader.IsValidBlock(b) {
		return fmt.Errorf(
			"invalid block received, [%d][%d]",
			b.PieceIdx, t.Downloader.BlockIdx(b))
	}

	blockLen, err := t.GetBlockLength(b.PieceIdx, t.Downloader.BlockIdx(b))
	if err != nil {
		return fmt.Errorf("error getting block length: %w", err)
	}

	// Validate length of the block data received
	if blockLen != len(blockData) {
		return fmt.Errorf("block len mismatch, expected:%d, got:%d", blockLen, len(blockData))
	}

	fmt.Printf("PIECE message received for: [%d][%d], from peer:%s\n", b.PieceIdx, t.Downloader.BlockIdx(b), p.Conn.RemoteAddr().String())

	// Mark as downloaded
	t.Downloader.Downloaded(b, blockData)

	if t.Downloader.IsDownloadComplete() {
		// Close connection to the peer
		return p.Conn.Close()
	}

	// Download incomplete
	return requestOnePiece(p, t.Downloader)
}

func parsePieceMsg(payload []byte) (*queue.Block, []byte) {
	pieceIdx := binary.BigEndian.Uint32(payload[0:4])
	blockOffset := binary.BigEndian.Uint32(payload[4:8])
	blockData := payload[8:]

	return queue.NewBlock(int(pieceIdx), int(blockOffset), len(blockData)), blockData
}
