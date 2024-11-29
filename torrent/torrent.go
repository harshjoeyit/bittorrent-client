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
}

func NewTorrent(decoded interface{}) *Torrent {
	return &Torrent{
		T: decoded,
	}
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

func (t *Torrent) GetInfoHash() ([20]byte, error) {
	infoHash := [20]byte{0}

	// type assert to map[string]interface{}
	torrentMap, ok := t.T.(map[string]interface{})
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

	// converting to string - not required
	// infoHashStr = hex.EncodeToString(infoHash[:])

	// save for future use
	t.InfoHash = infoHash

	return infoHash, nil
}

// GetFileLength returns file length (in bytes)
func (t *Torrent) GetFileLength() (int64, error) {
	if t.FileLength > 0 {
		return t.FileLength, nil
	}

	var fileLength int64

	// type assert to map[string]interface{}
	torrentMap, ok := t.T.(map[string]interface{})
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

	// update field in torrent instance for future use
	t.FileLength = fileLength

	return fileLength, nil
}

// GetPiecesCount returns number of pieces for the torrent
func (t *Torrent) GetPiecesCount() (int, error) {
	if t.PiecesCount > 0 {
		return t.PiecesCount, nil
	}

	var c int

	// type assert to map[string]interface{}
	torrentMap, ok := t.T.(map[string]interface{})
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

	c = len(piecesStr) / 20
	t.PiecesCount = c

	return c, nil
}

func (t *Torrent) GetPieceLength() (int, error) {
	if t.PieceLength > 0 {
		return t.PieceLength, nil
	}

	var l int

	// type assert to map[string]interface{}
	torrentMap, ok := t.T.(map[string]interface{})
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

	l = int(lenInt)
	t.PieceLength = l

	return l, nil
}

// GetPieceLengthAtPosition returns the length of a piece at a given
// index (in bytes)
// A file is divided into pieces of equal length except the last piece which may or
// may not be of same length as other pieces
func (t *Torrent) GetPieceLengthAtPosition(pieceIdx int) (int, error) {
	if pieceIdx < 0 {
		return 0, fmt.Errorf("piece index cannot be < 0")
	}

	fileLength, err := t.GetFileLength()
	if err != nil {
		return 0, fmt.Errorf("error getting file length: %w", err)
	}

	pieceLength, err := t.GetPieceLength()
	if err != nil {
		return 0, fmt.Errorf("error getting pieces count: %w", err)
	}

	lastPieceIdx := int(fileLength / int64(pieceLength))
	if pieceIdx > lastPieceIdx {
		return 0, fmt.Errorf("invalid piece index, exceeds number of pieces")
	}

	lastPieceLength := int(fileLength % int64(pieceLength))

	if pieceIdx == lastPieceIdx {
		return lastPieceLength, nil
	}

	return pieceLength, nil
}

const defaultBlockLength int = 16384 // 16KB

// GetBlocksCount returns the number of block into which the piece at pieceIdx
// can be divided into
func (t *Torrent) GetBlocksCount(pieceIdx int) (int, error) {
	pieceLength, err := t.GetPieceLengthAtPosition(pieceIdx)
	if err != nil {
		return 0, fmt.Errorf("error getting piece length at postion: %w", err)
	}

	return int(math.Ceil(float64(pieceLength) / float64(defaultBlockLength))), nil
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
		return 0, fmt.Errorf("invalid block index, exceeds number of blocks")
	}

	// piece length
	pieceLength, err := t.GetPieceLengthAtPosition(pieceIdx)
	if err != nil {
		return 0, fmt.Errorf("error getting piece length: %w", err)
	}

	lastBlockLength := pieceLength % defaultBlockLength

	if blockIdx == lastBlockIdx {
		return lastBlockLength, nil
	}

	return defaultBlockLength, nil
}
