package torrent

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSplitTorrentDataIntoFiles(t *testing.T) {
	// Create a temporary sparse file for testing
	tempDir := t.TempDir()
	srcFilePath := filepath.Join(tempDir, "sparse_file")
	src, err := os.Create(srcFilePath)
	if err != nil {
		t.Fatalf("Failed to create sparse file: %v", err)
	}
	defer src.Close()

	// Write dummy data to the sparse file
	data := []byte("File1ContentFile2Content")
	_, err = src.Write(data)
	if err != nil {
		t.Fatalf("Failed to write to sparse file: %v", err)
	}

	// Define FileMeta for two files
	files := []*FileMeta{
		{
			Path:   []string{"file1.txt"},
			Length: int64(len("File1Content")),
		},
		{
			Path:   []string{"file2.txt"},
			Length: int64(len("File2Content")),
		},
	}

	torr := &Torrent{
		Downloader: &Downloader{
			f:                 src,
			writesCh:          make(chan *Piece),
			writesCompletedCh: make(chan struct{}),
		},
		Files: files,
	}

	// Call the function to test
	err = torr.SplitTorrentDataIntoFiles()
	if err != nil {
		t.Fatalf("SplitTorrentDataIntoFiles failed: %v", err)
	}

	// Validate the created files
	for i, file := range torr.Files {
		expectedContent := data[int64(i)*file.Length : int64(i+1)*file.Length]
		content, err := os.ReadFile(file.Path[len(file.Path)-1])
		if err != nil {
			t.Fatalf("Failed to read output file %s: %v", file.Path[len(file.Path)-1], err)
		}

		// Ensure file cleanup after test
		filename := file.Path[len(file.Path)-1]
		defer os.Remove(filename)

		if !bytes.Equal(content, expectedContent) {
			t.Errorf("Content mismatch in file %s, expected %s, got %s",
				file.Path[len(file.Path)-1], string(expectedContent), string(content))
		}
	}

	// Check if the sparse file still exists (optional)
	_, err = os.Stat(srcFilePath)
	if err != nil {
		t.Errorf("Sparse file was unexpectedly deleted: %v", err)
	}

	//
}
