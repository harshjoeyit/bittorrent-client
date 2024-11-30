package queue

// Block is unit which can be requested over from a peer
// in a request message
type Block struct {
	pieceIdx    int
	blockOffset int // Offset/Index of the Block for piece denoted by PieceIdx
	blockLength int // Length of block (in bytes)
}

func NewBlock(pieceIdx, blockOffset, blockLength int) Block {
	return Block{
		pieceIdx:    pieceIdx,
		blockOffset: blockLength,
		blockLength: blockLength,
	}
}
