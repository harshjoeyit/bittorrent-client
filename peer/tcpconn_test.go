package peer

import (
	"encoding/hex"
	"testing"
)

func TestIsHandshakeMessageValid(t *testing.T) {
	var testCases = map[string]struct {
		msg      []byte
		infoHash string
	}{
		"first": {
			msg:      []byte{19, 66, 105, 116, 84, 111, 114, 114, 101, 110, 116, 32, 112, 114, 111, 116, 111, 99, 111, 108, 0, 0, 0, 0, 0, 16, 0, 5, 201, 225, 87, 99, 247, 34, 242, 62, 152, 162, 157, 236, 223, 174, 52, 27, 152, 213, 48, 86, 45, 85, 84, 51, 54, 48, 87, 45, 64, 184, 237, 91, 100, 59, 120, 50, 159, 4, 86, 234},
			infoHash: "c9e15763f722f23e98a29decdfae341b98d53056",
		},
		"second": {
			msg:      []byte{19, 66, 105, 116, 84, 111, 114, 114, 101, 110, 116, 32, 112, 114, 111, 116, 111, 99, 111, 108, 0, 0, 0, 0, 0, 16, 0, 5, 201, 225, 87, 99, 247, 34, 242, 62, 152, 162, 157, 236, 223, 174, 52, 27, 152, 213, 48, 86, 45, 84, 82, 51, 48, 48, 48, 45, 104, 106, 56, 113, 53, 56, 53, 48, 97, 103, 120, 120},
			infoHash: "c9e15763f722f23e98a29decdfae341b98d53056",
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			// t.Parallel()
			infoHash, err := hex.DecodeString(test.infoHash)
			if err != nil {
				t.Errorf("failed to decode hex: %v", err)
			}

			err = IsHandshakeMessageValid(test.msg, [20]byte(infoHash))
			if err != nil {
				t.Errorf("invalid handshake: %v", err)
			}
		})
	}
}
