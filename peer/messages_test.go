package peer

import (
	"encoding/binary"
	"testing"
)

func TestParseChokeMessage(t *testing.T) {
	m := BuildChokeMessage()

	l := binary.BigEndian.Uint32(m[0:4])
	if l != 1 {
		t.Errorf("message length mismatch, expected %d, got %d", 1, l)
	}

	if m[4] != byte(Choke) {
		t.Errorf("messageID mismatch, expected: %d, got %d", Choke, m[4])
	}
}

func TestParseUnchokeMessage(t *testing.T) {
	m := BuildUnchokeMessage()

	l := binary.BigEndian.Uint32(m[0:4])
	if l != 1 {
		t.Errorf("message length mismatch, expected %d, got %d", 1, l)
	}

	if m[4] != byte(Unchoke) {
		t.Errorf("messageID mismatch, expected: %d, got %d", Unchoke, m[4])
	}
}
