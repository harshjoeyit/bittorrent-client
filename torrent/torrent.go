package torrent

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math"
	"os"

	"github.com/jackpal/bencode-go"
)

type Torrent struct {
	Name        string      // Name is directory name where the files are to be saved
	Files       []*FileMeta // Meta info of files to be downloaded
	Decoded     interface{} // Decoded torrent
	InfoHash    [20]byte
	FileLength  int64
	PiecesCount int
	PieceLength int
	PieceHash   [][20]byte // sha-1 hash for all the pieces
	Downloader  *Downloader
}

func NewTorrent(decoded interface{}) (t *Torrent, err error) {
	t = &Torrent{
		Decoded: decoded,
	}

	// Check validity of torrent by per-calculating fields
	// at the time of creation of NewTorrent. These fields will
	// be needed in further steps
	t.Name, err = getName(decoded)
	if err != nil {
		return nil, fmt.Errorf("error getting name: %v", err)
	}

	t.InfoHash, err = getInfoHash(decoded)
	if err != nil {
		return nil, fmt.Errorf("error getting info hash: %v", err)
	}

	t.Files, err = getFiles(decoded)
	if err != nil {
		return nil, fmt.Errorf("error getting files: %v", err)
	}

	t.FileLength = getFileLength(t.Files)

	t.PiecesCount, t.PieceHash, err = getPiecesHashAndCount(decoded)
	if err != nil {
		return nil, fmt.Errorf("error getting piece hashes and count: %v", err)
	}
	log.Println("total pieces:", t.PiecesCount)

	t.PieceLength, err = getPieceLength(decoded)
	if err != nil {
		return nil, fmt.Errorf("error getting pieces length: %v", err)
	}
	log.Printf("piece length: %d bytes (%d KB)\n", t.PieceLength, t.PieceLength/1024)
	blocksPerPiece, _ := t.GetBlocksCount(0)
	log.Println("blocks per piece >=", blocksPerPiece)

	t.Downloader, err = NewDownloader(t)
	if err != nil {
		return nil, fmt.Errorf("error creating new downloader: %v", err)
	}

	return t, nil
}

// FileMeta represent each file in a multi-file torrent
// In case of a single file torrent, the FileMeta represents itself
type FileMeta struct {
	// Path is location on disk relative to parent torrent folder
	// For e.g. ["folder1", "images", "pic.jpg"]
	Path []string

	// Length is file size in bytes
	Length int64
}

// GetAnnounceUrl extracts and returns annouce url (tracker url)
// from torrent
func (t *Torrent) GetAnnounceUrl() (string, error) {
	// type assert to map[string]interface{}
	torrentMap, ok := t.Decoded.(map[string]interface{})
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

func getName(decoded interface{}) (string, error) {
	// type assert to map[string]interface{}
	torrentMap, ok := decoded.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("decoded data is not a map")
	}

	// check if info field exists
	info, ok := torrentMap["info"]
	if !ok {
		return "", fmt.Errorf("'info' field does not exist")
	}

	// type assert
	infoMap, ok := info.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("'info' field not a map")
	}

	name, ok := infoMap["name"]
	if !ok {
		return "", fmt.Errorf("'info.name' field does not exist")
	}

	nameStr, ok := name.(string)
	if !ok {
		return "", fmt.Errorf("'info.name' field not a string")
	}

	log.Println("Name:", nameStr)

	return nameStr, nil
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
func getFileLength(files []*FileMeta) int64 {
	var fileLength int64

	for _, file := range files {
		fileLength += file.Length
	}

	log.Printf(
		"total file length: %d bytes (%d KB), (%0.2f MB)\n",
		fileLength,
		fileLength/1024,
		float64(fileLength)/1024/1024,
	)

	return fileLength
}

func getFiles(decoded interface{}) ([]*FileMeta, error) {
	// type assert to map[string]interface{}
	torrentMap, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("decoded data is not a map")
	}

	// check if info field exists
	info, ok := torrentMap["info"]
	if !ok {
		return nil, fmt.Errorf("'info' field does not exist")
	}

	// type assert
	infoMap, ok := info.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("'info' field not a map")
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

	// Single file torrent
	// check if 'length' key in present in dict, if yes, it's a single file
	length, ok := infoMap["length"]
	if ok {
		// file length (size)
		fileLength, ok := length.(int64)
		if !ok {
			return nil, fmt.Errorf("info.length field is not int64")
		}

		file := &FileMeta{
			Path:   nil,
			Length: fileLength,
		}

		return []*FileMeta{file}, nil
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

	// Multi-file torrent
	f, ok := infoMap["files"]
	if !ok {
		return nil, fmt.Errorf("info.files field is not exist")
	}

	// type asset - fl should be a slice
	fl, ok := f.([]interface{})
	if !ok {
		return nil, fmt.Errorf("info.files field is not a list")
	}

	log.Printf("multi file torrent\n")

	var files []*FileMeta

	for i, f := range fl {
		// Each file should be a map[string]{}
		fileMap, ok := f.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("info.files[%d] value is not of type map[string]interface{}", i)
		}

		file := &FileMeta{}

		// Check if length field exists
		length, ok := fileMap["length"]
		if ok {
			fileLength, ok := length.(int64)
			if !ok {
				return nil, fmt.Errorf("info.files[%d].length field is not int64", i)
			}
			file.Length = fileLength
		}

		// Check if path field exists
		path, ok := fileMap["path"]
		if ok {
			filepath, ok := path.([]interface{})
			if !ok {
				return nil, fmt.Errorf("inf.files[%d].path field is not a list", i)
			}

			// type cast to string and append to file.Path
			file.Path = make([]string, 0)
			for j, fp := range filepath {
				fpstr, ok := fp.(string)
				if !ok {
					return nil, fmt.Errorf("inf.files[%d].path[%d] field is not a string", i, j)
				}
				file.Path = append(file.Path, fpstr)
			}

		}

		files = append(files, file)
	}

	// Print files
	for i, file := range files {
		log.Printf("file[%d], length: %d, path: %v\n", i, file.Length, file.Path)
	}

	return files, nil
}

// GetPiecesCount returns number of pieces for the torrent
func getPiecesHashAndCount(decoded interface{}) (int, [][20]byte, error) {
	var c int
	var pieceHashes [][20]byte

	// type assert to map[string]interface{}
	torrentMap, ok := decoded.(map[string]interface{})
	if !ok {
		return c, pieceHashes, fmt.Errorf("decoded data is not a map")
	}

	// check if info field exists
	info, ok := torrentMap["info"]
	if !ok {
		return c, pieceHashes, fmt.Errorf("'info' field does not exist")
	}

	// type assert
	infoMap, ok := info.(map[string]interface{})
	if !ok {
		return c, pieceHashes, fmt.Errorf("'info' field not a map")
	}

	pieces, ok := infoMap["pieces"]
	if !ok {
		return c, pieceHashes, fmt.Errorf("'pieces' field does not exist in 'info'")
	}

	// type assert
	piecesStr, ok := pieces.(string)
	if !ok {
		return c, pieceHashes, fmt.Errorf("'pieces' field not a string")
	}

	// since pieces is concatination of 20 bytes sha1 hashes,
	// it should be divisible by 20
	if len(piecesStr)%20 != 0 {
		return c, pieceHashes, fmt.Errorf("length of pieces is not divisible by 20")
	}

	// Preallocate slice for piece hashes for efficiency
	pieceHashes = make([][20]byte, 0, len(piecesStr)/20)

	for i := 0; i < len(piecesStr); i = i + 20 {
		var h [20]byte
		copy(h[:], piecesStr[i:i+20])
		pieceHashes = append(pieceHashes, h)
	}

	return len(piecesStr) / 20, pieceHashes, nil
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

// SplitTorrentDataIntoFiles splits the downloaded torrent.data files
// into respective files as defined in FileMeta
func (t *Torrent) SplitTorrentDataIntoFiles() error {
	if !t.Downloader.IsDownloadComplete() {
		return fmt.Errorf("download incomplete")
	}

	src := t.Downloader.f
	defer src.Close()

	// Global offset tracks the position in the sparse file
	globalOffset := int64(0)

	for _, file := range t.Files {
		filename := file.Path[len(file.Path)-1]

		// Open the target file for writing
		dst, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("error creating file %s: %v", filename, err)
		}
		defer dst.Close()

		// Copy the file's data from the sparse file
		_, err = src.Seek(globalOffset, io.SeekStart)
		if err != nil {
			return fmt.Errorf("error seeking in sparse file: %v", err)
		}

		// Limit the reader to the file's size
		_, err = io.CopyN(dst, src, file.Length)
		if err != nil && err != io.EOF {
			return fmt.Errorf("error copying data to file %s: %v", filename, err)
		}

		// Update the global offset
		globalOffset += file.Length
	}

	// todo: Delete the sparse file

	return nil
}
