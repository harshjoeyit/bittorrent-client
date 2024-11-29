package peer

import (
	"reflect"
	"testing"
)

func TestBitfieldMsgHandler(t *testing.T) {
	var testCases = map[string]struct {
		payload         []byte
		expectedIndices []int
	}{
		"3 bytes bitfield": {
			payload:         []byte{80, 48, 67},
			expectedIndices: []int{3, 1, 11, 10, 23, 22, 17},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			gotIndices := bitfieldMsgHandler(test.payload)
			if !reflect.DeepEqual(gotIndices, test.expectedIndices) {
				t.Errorf("expected: %v, got: %v", test.expectedIndices, gotIndices)
			}
		})
	}
}
