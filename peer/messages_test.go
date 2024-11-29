package peer

import (
	"encoding/binary"
	"fmt"
	"reflect"
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

func TestBuildBitFieldMessage(t *testing.T) {
	var testCases = map[string]struct {
		pieces           []bool
		expectedBitfield []byte
	}{
		"1 byte bitfield": {
			pieces:           []bool{false, true, false, true, false},
			expectedBitfield: []byte{80},
		},
		"2 byte bitfield": {
			pieces:           []bool{false, false, false, false, false, false, false, false, false, false, true, true},
			expectedBitfield: []byte{0, 48},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			msg, err := BuildBitFieldMessage(test.pieces)
			fmt.Println("msg: ", msg)

			if err != nil {
				t.Errorf("failed to build message: %v", err)
			}

			// Validate message length
			msgLen := binary.BigEndian.Uint32(msg[:4])
			if msgLen != uint32(len(msg[4:])) {
				t.Errorf("invalid message length, expected: %d, got: %d", msgLen, len(msg[4:]))
			}

			// Validate message ID
			if msg[4] != byte(Bitfield) {
				t.Errorf("invalid message ID, expected: %d, got: %d", Bitfield, msg[4])
			}

			// Validate bitfield
			if !reflect.DeepEqual(msg[5:], test.expectedBitfield) {
				t.Errorf("bitfield payload mismatch, expected: %v, got: %v", test.expectedBitfield, msg[5:])
			}
		})
	}
}
