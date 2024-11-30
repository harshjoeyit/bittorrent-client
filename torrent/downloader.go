package torrent

import "fmt"

type Downloader struct {
	requestedBlocks  [][]bool
	downloadedBlocks [][]bool
}

func NewDowloader(t *Torrent) (*Downloader, error) {
	d := &Downloader{
		downloadedBlocks: make([][]bool, t.PiecesCount),
		requestedBlocks:  make([][]bool, t.PiecesCount),
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

func (d *Downloader) Downloaded(pieceIdx, blockOffset int) {
	d.downloadedBlocks[pieceIdx][blockOffset] = true
}

func (d *Downloader) Requested(pieceIdx, blockOffset int) {
	d.requestedBlocks[pieceIdx][blockOffset] = true
}

func (d *Downloader) IsRequested(pieceIdx, blockOffset int) bool {
	return d.requestedBlocks[pieceIdx][blockOffset]
}
