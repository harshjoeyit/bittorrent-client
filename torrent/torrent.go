package torrent

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"math"

	"github.com/jackpal/bencode-go"
)

type Torrent struct {
	T           interface{} // decoded torrent
	InfoHash    [20]byte
	FileLength  int64 // to avoid re-computation
	PiecesCount int   // number of pieces (to avoid re-computation)
	PieceLength int   // to avoid re-computation
	Downloader  *Downloader
}

func NewTorrent(decoded interface{}) (t *Torrent, err error) {
	t = &Torrent{
		T: decoded,
	}

	// Check validity of torrent by per-calculating fields
	// at the time of creation of NewTorrent. These fields will
	// be needed in further steps

	t.InfoHash, err = getInfoHash(decoded)
	if err != nil {
		return nil, fmt.Errorf("error getting info hash: %v", err)
	}

	t.FileLength, err = getFileLength(decoded)
	if err != nil {
		return nil, fmt.Errorf("error getting file length: %v", err)
	}

	t.PiecesCount, err = getPiecesCount(decoded)
	if err != nil {
		return nil, fmt.Errorf("error getting pieces count: %v", err)
	}
	log.Println("total pieces:", t.PiecesCount)

	t.PieceLength, err = getPieceLength(decoded)
	if err != nil {
		return nil, fmt.Errorf("error getting pieces length: %v", err)
	}
	log.Println("piece length:", t.PieceLength)
	blocksPerPiece, _ := t.GetBlocksCount(0)
	log.Println("blocks per piece >=", blocksPerPiece)

	t.Downloader, err = NewDownloader(t)
	if err != nil {
		return nil, fmt.Errorf("error creating new downloader: %v", err)
	}

	return t, nil
}

// GetAnnounceUrl extracts and returns annouce url (tracker url)
// from torrent
func (t *Torrent) GetAnnounceUrl() (string, error) {
	// type assert to map[string]interface{}
	torrentMap, ok := t.T.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("decoded data is not a map")
	}

	// check if announce field exists
	announce, ok := torrentMap["announce"]
	if !ok {
		return "", fmt.Errorf("'announce' field does not exist")
	}

	// type assert to string
	announceUrl, ok := announce.(string)
	if !ok {
		return "", fmt.Errorf("'announce' field is not a string")
	}

	return announceUrl, nil
}

func getInfoHash(decoded interface{}) ([20]byte, error) {
	infoHash := [20]byte{0}

	// type assert to map[string]interface{}
	torrentMap, ok := decoded.(map[string]interface{})
	if !ok {
		return infoHash, fmt.Errorf("decoded data is not a map")
	}

	// check if info field exists
	info, ok := torrentMap["info"]
	if !ok {
		return infoHash, fmt.Errorf("'info' field does not exist")
	}

	// type assert
	infoMap, ok := info.(map[string]interface{})
	if !ok {
		return infoHash, fmt.Errorf("'info' field not a map")
	}

	// buffer to store bencoded 'info'
	var buf bytes.Buffer

	// to bencode
	err := bencode.Marshal(&buf, infoMap)
	if err != nil {
		return infoHash, fmt.Errorf("info could not be bencoded")
	}

	infoHash = sha1.Sum(buf.Bytes())

	// prints 40 character hexadecimal string
	log.Printf("info_hash: %s\n", hex.EncodeToString(infoHash[:]))

	return infoHash, nil
}

// GetFileLength returns file length (in bytes)
func getFileLength(decoded interface{}) (int64, error) {
	var fileLength int64

	// type assert to map[string]interface{}
	torrentMap, ok := decoded.(map[string]interface{})
	if !ok {
		return fileLength, fmt.Errorf("decoded data is not a map")
	}

	// check if info field exists
	info, ok := torrentMap["info"]
	if !ok {
		return fileLength, fmt.Errorf("'info' field does not exist")
	}

	// type assert
	infoMap, ok := info.(map[string]interface{})
	if !ok {
		return fileLength, fmt.Errorf("'info' field not a map")
	}

	/*
		For a single-file torrent: length (value in the info dictionary) gives
		the total length.

		{
			"announce": "http://tracker.example.com:8080/announce",
			"info": {
				"length": 1024000
			},
		}
	*/

	// check if 'length' key in present in dict, if yes, it's a single file

	length, ok := infoMap["length"]
	if ok {
		fileLength, ok := length.(int64)
		if !ok {
			return fileLength, fmt.Errorf("info.length field is not int64")
		}

		log.Printf("single file torrent, length: %d bytes", fileLength)
		return fileLength, nil
	}

	// !ok - multiple files

	/*
		For a multi-file torrent: Add up the lengths of all files listed in the
		files array under the info dictionary.

		{
			"announce": "http://tracker.example.com:8080/announce",
			"info": {
				"name": "example_folder",
				"piece length": 262144,
				"pieces": "abcd1234efgh5678...",
				"files": [
					{
						"length": 512000,
						"path": ["file1.txt"]
					},
					{
						"length": 1024000,
						"path": ["subfolder2", "file2.txt"]
					}
				]
			}
		}
	*/

	files, ok := infoMap["files"]
	if !ok {
		return fileLength, fmt.Errorf("info.files field is not exist")
	}

	// type asset - files should be a slice (list)
	fileList, ok := files.([]interface{})
	if !ok {
		return fileLength, fmt.Errorf("info.files field is not a list")
	}

	log.Printf("multi file torrent\n")
	for i, file := range fileList {
		// each file should be a map[string]{}
		fileMap, ok := file.(map[string]interface{})
		if !ok {
			return fileLength, fmt.Errorf("info.files.file value is not of type map[string]interface{}")
		}

		// check if length field exists
		length, ok := fileMap["length"]
		if ok {
			l, ok := length.(int64)
			if !ok {
				return fileLength, fmt.Errorf("info.files.file[%d].length field is not int64", i)
			}

			// Add to the total length
			log.Printf("file[%d], length: %d bytes\n", i, l)
			fileLength += l
		}
	}

	log.Printf("total length of multi-file: %d bytes\n", fileLength)

	return fileLength, nil
}

// GetPiecesCount returns number of pieces for the torrent
func getPiecesCount(decoded interface{}) (int, error) {
	var c int

	// type assert to map[string]interface{}
	torrentMap, ok := decoded.(map[string]interface{})
	if !ok {
		return c, fmt.Errorf("decoded data is not a map")
	}

	// check if info field exists
	info, ok := torrentMap["info"]
	if !ok {
		return c, fmt.Errorf("'info' field does not exist")
	}

	// type assert
	infoMap, ok := info.(map[string]interface{})
	if !ok {
		return c, fmt.Errorf("'info' field not a map")
	}

	pieces, ok := infoMap["pieces"]
	if !ok {
		return c, fmt.Errorf("'pieces' field does not exist in 'info'")
	}

	// type assert
	piecesStr, ok := pieces.(string)
	if !ok {
		return c, fmt.Errorf("'pieces' field not a string")
	}

	// since pieces is concatination of 20 bytes sha1 hashes,
	// it should be divisible by 20
	if len(piecesStr)%20 != 0 {
		return c, fmt.Errorf("length of pieces is not divisible by 20")
	}

	return len(piecesStr) / 20, nil
}

func getPieceLength(decoded interface{}) (int, error) {
	var l int

	// type assert to map[string]interface{}
	torrentMap, ok := decoded.(map[string]interface{})
	if !ok {
		return l, fmt.Errorf("decoded data is not a map")
	}

	// check if info field exists
	info, ok := torrentMap["info"]
	if !ok {
		return l, fmt.Errorf("'info' field does not exist")
	}

	// type assert
	infoMap, ok := info.(map[string]interface{})
	if !ok {
		return l, fmt.Errorf("'info' field not a map")
	}

	len, ok := infoMap["piece length"]
	if !ok {
		return l, fmt.Errorf("'piece length' field does not exist in 'info'")
	}

	lenInt, ok := len.(int64)
	if !ok {
		return l, fmt.Errorf("'piece length' field is not a int")
	}

	return int(lenInt), nil
}

// GetPieceLengthAtPosition returns the length of a piece at a given
// index (in bytes)
// A file is divided into pieces of equal length except the last piece which may or
// may not be of same length as other pieces
func (t *Torrent) GetPieceLengthAtPosition(pieceIdx int) (int, error) {
	if pieceIdx < 0 {
		return 0, fmt.Errorf("piece index cannot be < 0")
	}

	lastPieceIdx := int(t.FileLength / int64(t.PieceLength))
	if pieceIdx > lastPieceIdx {
		return 0, fmt.Errorf("invalid piece index %d, exceeds valid range [0, %d]", pieceIdx, lastPieceIdx)
	}

	lastPieceLength := int(t.FileLength % int64(t.PieceLength))

	if pieceIdx == lastPieceIdx {
		return lastPieceLength, nil
	}

	return t.PieceLength, nil
}

const DefaultBlockLength int = 16384 // 16KB

// GetBlocksCount returns the number of block into which the piece at pieceIdx
// can be divided into
func (t *Torrent) GetBlocksCount(pieceIdx int) (int, error) {
	pieceLength, err := t.GetPieceLengthAtPosition(pieceIdx)
	if err != nil {
		return 0, fmt.Errorf("error getting piece length at postion: %w", err)
	}

	return int(math.Ceil(float64(pieceLength) / float64(DefaultBlockLength))), nil
}

// A piece is divided into blocks of equal length except the last block
// which may or may not be of same length as others
func (t *Torrent) GetBlockLength(pieceIdx, blockIdx int) (int, error) {
	blocks, err := t.GetBlocksCount(pieceIdx)
	if err != nil {
		return 0, fmt.Errorf("error getting blocks count: %w", err)
	}

	lastBlockIdx := blocks - 1
	if blockIdx > lastBlockIdx {
		return 0, fmt.Errorf("invalid block index %d, exceeds valid range [0, %d]", blockIdx, lastBlockIdx)
	}

	// piece length
	pieceLength, err := t.GetPieceLengthAtPosition(pieceIdx)
	if err != nil {
		return 0, fmt.Errorf("error getting piece length: %w", err)
	}

	lastBlockLength := pieceLength % DefaultBlockLength

	// If pieceLength is multiple of DefaultBlockLength, then lastBlockLength is 0
	// In this case last blocks is no special from other blocks
	// handling this properly and setting it to DefaultBlockLength
	if lastBlockLength == 0 && pieceLength > 0 {
		lastBlockLength = DefaultBlockLength
	}

	if blockIdx == lastBlockIdx {
		return lastBlockLength, nil
	}

	return DefaultBlockLength, nil
}
