package torrent

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/jackpal/bencode-go"
)

type Torrent struct {
	T        interface{} // decoded torrent
	InfoHash [20]byte
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

// getFileSize returns file size (in bytes)
func (t *Torrent) GetFileSize() (int64, error) {
	var fileSize int64

	// type assert to map[string]interface{}
	torrentMap, ok := t.T.(map[string]interface{})
	if !ok {
		return fileSize, fmt.Errorf("decoded data is not a map")
	}

	// check if info field exists
	info, ok := torrentMap["info"]
	if !ok {
		return fileSize, fmt.Errorf("'info' field does not exist")
	}

	// type assert
	infoMap, ok := info.(map[string]interface{})
	if !ok {
		return fileSize, fmt.Errorf("'info' field not a map")
	}

	/*
		For a single-file torrent: length (value in the info dictionary) gives
		the total size.

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
		fileSize, ok := length.(int64)
		if !ok {
			return fileSize, fmt.Errorf("info.length field is not int64")
		}

		return fileSize, nil
	}

	// !ok - multiple files

	/*
		For a multi-file torrent: Add up the sizes of all files listed in the
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
		return fileSize, fmt.Errorf("info.files field is not exist")
	}

	// type asset - files should be a slice (list)
	fileList, ok := files.([]interface{})
	if !ok {
		return fileSize, fmt.Errorf("info.files field is not a list")
	}

	for i, file := range fileList {
		// each file should be a map[string]{}
		fileMap, ok := file.(map[string]interface{})
		if !ok {
			return fileSize, fmt.Errorf("info.files.file value is not of type map[string]interface{}")
		}

		// check if length field exists
		length, ok := fileMap["length"]
		if ok {
			size, ok := length.(int64)
			if !ok {
				return fileSize, fmt.Errorf("info.files.file[%d].length field is not int64", i)
			}

			// add to the total size
			fileSize += size
		}
	}

	return fileSize, nil
}
