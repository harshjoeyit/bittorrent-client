package queue

// Block is unit which can be requested over from a peer
// in a request message
type Block struct {
	PieceIdx    int
	BlockOffset int // Offset of the Block for piece denoted by PieceIdx
	BlockLength int // Length of block (in bytes)
}

func NewBlock(pieceIdx, blockOffset, blockLength int) *Block {
	return &Block{
		PieceIdx:    pieceIdx,
		BlockOffset: blockOffset,
		BlockLength: blockLength,
	}
}
