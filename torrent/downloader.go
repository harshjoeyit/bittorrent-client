package torrent

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"log"
	"my-bittorrent/queue"
	"os"
	"path/filepath"
	"sync"
)

const defaultWriteChanBuffer int = 10
const downloadFolder string = "./downloads"
const torrentSparseFileName string = "torrent.data"

type Downloader struct {
	requestedBlocks      [][]bool
	rbmu                 sync.Mutex // To synchronize access to requestedBlocks
	downloadedBlocks     [][]bool
	downloadedBlocksData [][][]byte    // To hold block data until it's persisted
	dbmu                 sync.Mutex    // To synchronize access to downloadedBlocks and downloadedBlocksData
	persistedPieces      []bool        // Pieces which have been saved to sparse file on disk
	f                    *os.File      // Torrent sparse file
	writesCh             chan *Piece   // To receive writes while making writes to file go-routine safe
	once                 sync.Once     // To close writeCh
	writesCompletedCh    chan struct{} // To notify when all writes are completed
	PieceHash            [][20]byte    // sha-1 hash for all the pieces
	PieceLength          int
}

// Piece is the smallest unit which can be written to a file on disk
type Piece struct {
	Data   []byte
	Offset int64
}

func NewDownloader(t *Torrent) (*Downloader, error) {
	f, err := createDownloadFile(t.Name)
	if err != nil {
		return nil, fmt.Errorf("error creating new file: %w", err)
	}

	d := &Downloader{
		requestedBlocks:      make([][]bool, t.PiecesCount),
		rbmu:                 sync.Mutex{},
		downloadedBlocks:     make([][]bool, t.PiecesCount),
		downloadedBlocksData: make([][][]byte, t.PiecesCount),
		dbmu:                 sync.Mutex{},
		persistedPieces:      make([]bool, t.PiecesCount),
		f:                    f,
		writesCh:             make(chan *Piece, defaultWriteChanBuffer),
		once:                 sync.Once{},
		writesCompletedCh:    make(chan struct{}),
		PieceHash:            t.PieceHash,
		PieceLength:          t.PieceLength,
	}

	for i := 0; i < t.PiecesCount; i++ {
		blocksCount, err := t.GetBlocksCount(i)
		if err != nil {
			return nil, fmt.Errorf("error getting blocks count for piece idx: %d, error: %w", i, err)
		}

		d.downloadedBlocks[i] = make([]bool, blocksCount)
		d.downloadedBlocksData[i] = make([][]byte, blocksCount)
		d.requestedBlocks[i] = make([]bool, blocksCount)
	}

	log.Printf("Downloader ready. blocks for first piece: %d, blocks for last: %d\n",
		len(d.downloadedBlocks[0]),
		len(d.downloadedBlocks[t.PiecesCount-1]),
	)

	return d, nil
}

func (d *Downloader) Start() {
	// Start receiving writes
	go func() {
		d.receiveWrites()
	}()

}

// createDownloadFile creates a new sparse file or truncates existing file with same name
// for saving pieces to disk
func createDownloadFile(torrentName string) (*os.File, error) {
	// Check if ./downloads folder exists, if not create one
	if err := existsIfNotCreateOne(downloadFolder); err != nil {
		return nil, fmt.Errorf("error creating download dir: %v", err)
	}

	// Check if folder for torrent exists inside download folder, if not create one
	dir := filepath.Join(downloadFolder, torrentName)
	if err := existsIfNotCreateOne(dir); err != nil {
		return nil, fmt.Errorf("error creating torrent dir: %v", err)
	}

	// Create/truncate sparse file
	path := filepath.Join(dir, torrentSparseFileName)
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("error creating torrent sparse file: %v", err)
	}

	return f, nil
}

// existsIfNotCreateOne checks if a directory exists, if not it creates one
// returns nil error on success
func existsIfNotCreateOne(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			// Create dir
			if creatDirErr := os.Mkdir(dir, 0755); creatDirErr != nil {
				return creatDirErr
			}
			// Dir created successfully
		}
		// Dir exists
	}
	return nil
}

func (d *Downloader) receiveWrites() {
	var piecesCount int

	// writesCh continues to Recieves pieces until channel is closed
	for p := range d.writesCh {
		pieceIdx := p.Offset / int64(d.PieceLength)

		fmt.Printf("write received for piece offset: %d, idx: %d\n", p.Offset, pieceIdx)

		if d.persistedPieces[pieceIdx] {
			fmt.Printf("skipping since already persisted: %d, idx: %d\n", p.Offset, pieceIdx)
			continue
		}

		// Checking overwrite possible in range where we're going to write
		if err := d.isOverwriting(p.Offset); err != nil {
			log.Printf("cannot write piece due to overwrite, offset: %d, idx: %d, err:%v\n", p.Offset, pieceIdx, err)
		}
		if err := d.isOverwriting(p.Offset + int64(len(p.Data)-1)); err != nil {
			log.Printf("cannot write piece due to overwrite, offset: %d, idx: %d, err:%v\n", p.Offset+int64(len(p.Data)-1), pieceIdx, err)
		}

		// No overwrites
		_, err := d.f.WriteAt(p.Data, p.Offset)
		if err != nil {
			log.Printf("error writing to file: %v", err)
		}

		piecesCount++

		// Sync to disk after every 10 pieces received
		if piecesCount == 10 {
			// Flush to disk
			if err := d.f.Sync(); err != nil {
				log.Printf("error flushing file, idx: %d, error: %v", pieceIdx, err)
				continue
			}

			log.Printf("flushed to disk at piece idx: %d\n", pieceIdx)
			// Reset piece count
			piecesCount = 0
		}

		// Mark piece as persisted
		d.persistedPieces[pieceIdx] = true

		fmt.Printf("write completed for piece offset: %d, idx: %d\n", p.Offset, pieceIdx)
	}

	// Notify that all the writes are completed
	d.writesCompletedCh <- struct{}{}
}

// isOverwriting checks if the current write is overwriting existing data
func (d *Downloader) isOverwriting(offset int64) error {
	// Read one byte at the offset
	buf := make([]byte, 1)
	_, err := d.f.ReadAt(buf, offset)

	// If the offset is beyond EOF, the file will give io.EOF error, which
	// we can ignore
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("error reading at offset: %d, error: %v", offset, err)
	}
	if buf[0] != 0 {
		return fmt.Errorf("data already present at offset: %d", offset)
	}

	return nil
}

// Downloaded should be called when a block is received
func (d *Downloader) Downloaded(b *queue.Block, blockData []byte) {
	if !d.IsValidBlock(b) {
		log.Printf("Invalid block: downloadedBlocks[%d][%d]", b.PieceIdx, d.BlockIdx(b))
		return
	}

	d.dbmu.Lock()
	defer d.dbmu.Unlock()

	d.downloadedBlocks[b.PieceIdx][d.BlockIdx(b)] = true
	d.downloadedBlocksData[b.PieceIdx][d.BlockIdx(b)] = blockData

	// Todo: 1. Check if all the blocks for the pieces are downloaded, if yes
	all := true
	for _, val := range d.downloadedBlocks[b.PieceIdx] {
		if !val {
			all = false
			break
		}
	}

	if all {
		// Piece download complete
		piece, err := d.constructPiece(b.PieceIdx)
		if err != nil {
			log.Printf("Error constructing the piece at idx: %d, error: %v\n", b.PieceIdx, err)
			return
		}

		// Verify piece hash for data integrity
		expectedHash := d.PieceHash[b.PieceIdx]
		gotHash := sha1.Sum(piece.Data)

		if bytes.Equal(expectedHash[:], gotHash[:]) {
			d.writesCh <- piece
		} else {
			log.Printf("Piece Hash mismatch for piece at idx: %d, expected: %v, got: %v\n", b.PieceIdx, expectedHash, gotHash)
			// Piece is corrupted and hence the blocks need to be downloaded again
			// Reset downloaded block data
			d.ResetPiece(b.PieceIdx)
		}
	}
}

func (d *Downloader) constructPiece(pieceIdx int) (*Piece, error) {
	buf := bytes.NewBuffer([]byte{})

	// Join blocks one after another to form a piece
	for _, block := range d.downloadedBlocksData[pieceIdx] {
		_, err := buf.Write(block)
		if err != nil {
			return nil, fmt.Errorf("error writing to buffer: %v", err)
		}
	}

	return &Piece{
		Data:   buf.Bytes(),
		Offset: int64(pieceIdx) * int64(d.PieceLength),
	}, nil
}

// Resets the downloaded data for the piece
func (d *Downloader) ResetPiece(pieceIdx int) {
	d.dbmu.Lock()
	// d.rbmu.Lock()
	defer d.dbmu.Unlock()
	// defer d.rbmu.Unlock()

	d.persistedPieces[pieceIdx] = false
	for i := 0; i < len(d.downloadedBlocks[pieceIdx]); i++ {
		d.downloadedBlocks[pieceIdx][i] = false
		d.downloadedBlocksData[pieceIdx][i] = nil
		// d.requestedBlocks[pieceIdx][i] = false
	}
}

// Requested should be called when a block is requested
func (d *Downloader) Requested(b *queue.Block) {
	if !d.IsValidBlock(b) {
		log.Printf("Invalid block: requestedBlocks[%d][%d]", b.PieceIdx, d.BlockIdx(b))
		return
	}

	d.rbmu.Lock()
	defer d.rbmu.Unlock()

	d.requestedBlocks[b.PieceIdx][d.BlockIdx(b)] = true
}

func (d *Downloader) IsNeeded(b *queue.Block) bool {
	if !d.IsValidBlock(b) {
		log.Printf("Invalid block: requestedBlocks[%d][%d]", b.PieceIdx, d.BlockIdx(b))
		return false // todo: sending false is not proper way to handle error
	}

	// check if all the pieces are requested
	_, req, tot := d.progressReport()

	d.rbmu.Lock()
	defer d.rbmu.Unlock()

	if req == tot {
		d.dbmu.Lock()
		// we need to re-request the pieces which are not downloaded yet
		for i := 0; i < len(d.requestedBlocks); i++ {
			for j := 0; j < len(d.requestedBlocks[i]); j++ {
				d.requestedBlocks = d.downloadedBlocks
			}
		}
		d.dbmu.Unlock()
	}

	return !d.requestedBlocks[b.PieceIdx][d.BlockIdx(b)]
}

// IsValidBlock checks if indices for a block are out of bounds
func (d *Downloader) IsValidBlock(b *queue.Block) bool {
	if b.PieceIdx < 0 || b.PieceIdx >= len(d.downloadedBlocks) ||
		d.BlockIdx(b) < 0 || d.BlockIdx(b) >= len(d.downloadedBlocks[b.PieceIdx]) {
		return false
	}

	return true
}

// BlockIdx returns index of the block from block offset
func (d *Downloader) BlockIdx(b *queue.Block) int {
	return b.BlockOffset / DefaultBlockLength
}

func (d *Downloader) PrintProgress() {
	down, req, tot := d.progressReport()
	downPercent := 100 * float64(down) / float64(tot)
	reqPercent := 100 * float64(req) / float64(tot)

	fmt.Println("--------- Download Progress ---------")
	fmt.Printf("downloaded: %d / %d (%.2f)\n", down, tot, downPercent)
	fmt.Printf("requested: %d / %d (%.2f)\n", req, tot, reqPercent)
	fmt.Println("-------------------------------------")
}

func (d *Downloader) IsDownloadComplete() bool {
	down, _, tot := d.progressReport()
	comp := down == tot

	if comp {
		// Sync file to disk
		if err := d.f.Sync(); err != nil {
			log.Printf("error saving the file after download complete: %v\n", err)
		}

		d.closeWriteCh()
	}

	return comp
}

// closeWriteCh prevents a panic from closing a closed channel
func (d *Downloader) closeWriteCh() {
	d.once.Do(func() {
		close(d.writesCh)
		fmt.Println("writesCh closed safely.")
	})
}

// progressReport is helper function which returns
// downloaded, requested and total blocks
func (d *Downloader) progressReport() (int, int, int) {
	var tot, req, down int

	d.rbmu.Lock()
	d.dbmu.Lock()
	defer d.rbmu.Unlock()
	defer d.dbmu.Unlock()

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
