package torrent

import (
	"fmt"
	"log"
	"my-bittorrent/queue"
	"sync"
)

type Downloader struct {
	requestedBlocks  [][]bool
	downloadedBlocks [][]bool
	mu               sync.Mutex // todo: better to use sync.RWLock
}

func NewDownloader(t *Torrent) (*Downloader, error) {
	d := &Downloader{
		downloadedBlocks: make([][]bool, t.PiecesCount),
		requestedBlocks:  make([][]bool, t.PiecesCount),
		mu:               sync.Mutex{},
	}

	for i := 0; i < t.PiecesCount; i++ {
		blocksCount, err := t.GetBlocksCount(i)
		if err != nil {
			return nil, fmt.Errorf("error getting blocks count for piece idx: %d, error: %w", i, err)
		}

		d.downloadedBlocks[i] = make([]bool, blocksCount)
		d.requestedBlocks[i] = make([]bool, blocksCount)
	}

	return d, nil
}

// Downloaded should be called when a block is received
func (d *Downloader) Downloaded(b *queue.Block) {
	if !isValidBlock(b.PieceIdx, blockIdx(b), d.downloadedBlocks) {
		log.Printf("Invalid block: downloadedBlocks[%d][%d]", b.PieceIdx, blockIdx(b))
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.downloadedBlocks[b.PieceIdx][blockIdx(b)] = true
}

// Requested should be called when a block is requested
func (d *Downloader) Requested(b *queue.Block) {
	if !isValidBlock(b.PieceIdx, blockIdx(b), d.downloadedBlocks) {
		log.Printf("Invalid block: requestedBlocks[%d][%d]", b.PieceIdx, blockIdx(b))
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.requestedBlocks[b.PieceIdx][blockIdx(b)] = true
}

// IsRequested should be called to check if piece has already
// been requested
func (d *Downloader) IsRequested(b *queue.Block) bool {
	if !isValidBlock(b.PieceIdx, blockIdx(b), d.downloadedBlocks) {
		log.Printf("Invalid block: requestedBlocks[%d][%d]", b.PieceIdx, blockIdx(b))
		return true // todo: sending true is not proper way to handle error
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	return d.requestedBlocks[b.PieceIdx][blockIdx(b)]
}

func (d *Downloader) IsNeeded(b *queue.Block) bool {
	if !isValidBlock(b.PieceIdx, blockIdx(b), d.downloadedBlocks) {
		log.Printf("Invalid block: requestedBlocks[%d][%d]", b.PieceIdx, blockIdx(b))
		return false // todo: sending false is not proper way to handle error
	}

	// check if all the pieces are requested
	_, req, tot := d.progressReport()

	d.mu.Unlock()
	defer d.mu.Unlock()

	if req == tot {
		// we need to re-request the pieces which are not downloaded yet
		for i := 0; i < len(d.requestedBlocks); i++ {
			for j := 0; j < len(d.requestedBlocks[i]); j++ {
				d.requestedBlocks = d.downloadedBlocks
			}
		}
	}

	return !d.requestedBlocks[b.PieceIdx][blockIdx(b)]
}

// isValidBlock checks if indices are out of bounds
func isValidBlock(pieceIdx, blockIdx int, a [][]bool) bool {
	if pieceIdx < 0 || pieceIdx >= len(a) ||
		blockIdx < 0 || blockIdx >= len(a[pieceIdx]) {
		return false
	}

	return true
}

// blockIdx returns index of the block from block offset
func blockIdx(b *queue.Block) int {
	return b.BlockOffset / DefaultBlockLength
}

func (d *Downloader) PrintProgress() {
	down, req, tot := d.progressReport()

	fmt.Println("--------- Download Progress ---------")
	fmt.Printf("downloaded: %.0f / %.0f (%.2f)\n", down, tot, down/tot)
	fmt.Printf("requested: %.0f / %.0f (%.2f)\n", req, tot, req/tot)
	fmt.Println("-------------------------------------")
}

func (d *Downloader) IsDownloadComplete() bool {
	down, _, tot := d.progressReport()

	return down == tot
}

// progressReport is helper function which returns
// downloaded, requested and total blocks
func (d *Downloader) progressReport() (float64, float64, float64) {
	var tot, req, down float64

	d.mu.Lock()
	defer d.mu.Unlock()

	for i := 0; i < len(d.downloadedBlocks); i++ {
		for j := 0; j < len(d.downloadedBlocks[i]); j++ {
			tot++

			if d.downloadedBlocks[i][j] {
				down++
			}

			if d.requestedBlocks[i][j] {
				req++
			}
		}
	}

	return down, req, tot
}
