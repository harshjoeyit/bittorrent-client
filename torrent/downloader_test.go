package torrent

import (
	"testing"
)

func TestNewDowloader(t *testing.T) {
	var testCases = map[string]struct {
		torrent *Torrent
	}{
		"first": {
			torrent: &Torrent{
				PiecesCount: 11,
				FileLength:  int64(10*256*1024 + 14*defaultBlockLength + 10), // 10 pieces and 15 blocks
				PieceLength: 256 * 1024,                                      // 256 KB
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			d, err := NewDowloader(test.torrent)
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
				if len(d.downloadedBlocks[i]) != test.torrent.PieceLength/defaultBlockLength {
					t.Errorf("downloadedBlocks[%d] len mismatch, expected: %d, got: %d", i, defaultBlockLength, len(d.downloadedBlocks[i]))
				}
			}

			// last piece
			lastPieceLen := test.torrent.FileLength % int64(test.torrent.PieceLength) % int64(test.torrent.PieceLength)
			blocksInLastPiece := lastPieceLen / int64(defaultBlockLength)

			if lastPieceLen%int64(defaultBlockLength) != 0 {
				blocksInLastPiece += 1
			}

			if int(blocksInLastPiece) != len(d.downloadedBlocks[test.torrent.PiecesCount-1]) {
				t.Errorf("last block count mismatch, expected: %d, got: %d", int(blocksInLastPiece), len(d.downloadedBlocks[test.torrent.PiecesCount-1]))
			}
		})
	}
}
