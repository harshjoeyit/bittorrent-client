package torrent

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestNewDowloader(t *testing.T) {
	var testCases = map[string]struct {
		torrent *Torrent
	}{
		"first": {
			torrent: &Torrent{
				PiecesCount: 11,
				FileLength:  int64(10*256*1024 + 14*DefaultBlockLength + 10), // 10 pieces and 15 blocks
				PieceLength: 256 * 1024,                                      // 256 KB
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			d, err := NewDownloader(test.torrent)
			if err != nil {
				t.Errorf("error creating new downloader: %v", err)
			}

			// validate d
			if len(d.downloadedBlocks) != test.torrent.PiecesCount {
				t.Errorf("downloadedBlocks len mismatch, expected: %d, got: %d", test.torrent.PiecesCount, len(d.downloadedBlocks))
			}
			if len(d.requestedBlocks) != test.torrent.PiecesCount {
				t.Errorf("downloadedBlocks len mismatch, expected: %d, got: %d", test.torrent.PiecesCount, len(d.requestedBlocks))
			}

			// all but last piece
			for i := 0; i < len(d.downloadedBlocks)-1; i++ {
				if len(d.downloadedBlocks[i]) != test.torrent.PieceLength/DefaultBlockLength {
					t.Errorf("downloadedBlocks[%d] len mismatch, expected: %d, got: %d", i, DefaultBlockLength, len(d.downloadedBlocks[i]))
				}
			}

			// last piece
			lastPieceLen := test.torrent.FileLength % int64(test.torrent.PieceLength) % int64(test.torrent.PieceLength)
			blocksInLastPiece := lastPieceLen / int64(DefaultBlockLength)

			if lastPieceLen%int64(DefaultBlockLength) != 0 {
				blocksInLastPiece += 1
			}

			if int(blocksInLastPiece) != len(d.downloadedBlocks[test.torrent.PiecesCount-1]) {
				t.Errorf("last block count mismatch, expected: %d, got: %d", int(blocksInLastPiece), len(d.downloadedBlocks[test.torrent.PiecesCount-1]))
			}
		})
	}
}

func TestCreateDownloadFile(t *testing.T) {
	var testCases = map[string]struct {
		torrent *Torrent
	}{
		"first": {
			torrent: &Torrent{
				Name: "Puppy Torrent",
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			f, err := createDownloadFile(test.torrent.Name)
			if err != nil {
				t.Error(err)
			}
			if f.Name() != filepath.Join(downloadFolder, test.torrent.Name, torrentSparseFileName) {
				t.Errorf("filename mismatch, expected: %s, got: %s", torrentSparseFileName, f.Name())
			}
		})
	}
}

func TestConstructPiece(t *testing.T) {
	tests := []struct {
		name             string
		pieceIdx         int
		pieceLength      int
		downloadedBlocks [][][]byte
		expectedData     []byte
		expectedOffset   int64
		expectedError    bool
	}{
		{
			name:        "valid piece with one block",
			pieceIdx:    0,
			pieceLength: 10,
			downloadedBlocks: [][][]byte{
				{[]byte("block1")},
			},
			expectedData:   []byte("block1"),
			expectedOffset: 0,
			expectedError:  false,
		},
		{
			name:        "valid piece with multiple blocks",
			pieceIdx:    1,
			pieceLength: 10,
			downloadedBlocks: [][][]byte{
				{[]byte("block1")},
				{[]byte("block2"), []byte("block3")},
			},
			expectedData:   []byte("block2block3"),
			expectedOffset: 10,
			expectedError:  false,
		},
		{
			name:        "empty piece",
			pieceIdx:    2,
			pieceLength: 10,
			downloadedBlocks: [][][]byte{
				{[]byte("block1")},
				{[]byte("block2"), []byte("block3")},
				{},
			},
			expectedData:   []byte{},
			expectedOffset: 20,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Downloader{
				PieceLength:          tt.pieceLength,
				downloadedBlocksData: tt.downloadedBlocks,
			}

			piece, err := d.constructPiece(tt.pieceIdx)

			// Check for error
			if (err != nil) != tt.expectedError {
				t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				return
			}

			// If no error, check the returned piece data and offset
			if err == nil {
				if !bytes.Equal(piece.Data, tt.expectedData) {
					t.Errorf("expected data: %v, got: %v", tt.expectedData, piece.Data)
				}
				if piece.Offset != tt.expectedOffset {
					t.Errorf("expected offset: %v, got: %v", tt.expectedOffset, piece.Offset)
				}
			}
		})
	}
}
