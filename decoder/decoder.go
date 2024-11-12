package decoder

import (
	"encoding/json"
	"fmt"
	"strconv"
	"unicode"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

func decodeBencodeHelper(bencodedString string) (interface{}, int, error) {

	// fmt.Printf("bencodedString: %s\n\n", bencodedString)

	if len(bencodedString) == 0 {
		return "", 0, nil
	}

	if unicode.IsDigit(rune(bencodedString[0])) {
		// string

		// check for empty string - 0:
		if bencodedString[0] == '0' {
			return "", 1, nil
		}

		var firstColonIndex int

		for i := 0; i < len(bencodedString); i++ {
			if bencodedString[i] == ':' {
				firstColonIndex = i
				break
			}
		}

		lengthStr := bencodedString[:firstColonIndex]

		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			return "", -1, err
		}

		// todo: check for length should not exceed int64

		// check for negetive length
		if length < 0 {
			return "", 1, fmt.Errorf("string length can not be a negative number")
		}

		return bencodedString[firstColonIndex+1 : firstColonIndex+1+length], firstColonIndex + length, nil

	} else if bencodedString[0] == 'i' {
		// integer
		var endIndex int

		for i := 1; i < len(bencodedString); i++ {
			if bencodedString[i] == 'e' {
				endIndex = i
				break
			}
		}

		// string between index 1 and endIndex is our integer
		intStr := bencodedString[1:endIndex]
		var err error

		if intVal, err := strconv.ParseInt(intStr, 10, 64); err == nil {
			// int64
			return intVal, endIndex, nil
		} else if intVal, err := strconv.ParseUint(intStr, 10, 64); err == nil {
			// uint64
			return intVal, endIndex, nil
		}

		return "", -1, err

	} else if bencodedString[0] == 'l' {
		var list []interface{}

		// empty list
		if len(bencodedString) >= 2 && bencodedString[1] == 'e' {
			return "", 1, nil
		}

		currIdx := 1
		// iterating over each element in the list

		for currIdx < len(bencodedString) && bencodedString[currIdx] != 'e' {
			decodedListItem, relativeEndIdx, decodingErr := decodeBencodeHelper(bencodedString[currIdx:])

			if decodingErr != nil {
				return "", -1, fmt.Errorf("error occured decoding: %c at: %d, %w", bencodedString[currIdx], currIdx, decodingErr)
			}

			list = append(list, decodedListItem)

			currIdx = currIdx + relativeEndIdx + 1
		}

		return list, currIdx, nil
	} else if bencodedString[0] == 'd' {
		// dictionary
		dict := map[string]interface{}{}

		// check for empty dictionary
		if len(bencodedString) >= 2 && bencodedString[1] == 'e' {
			return "", 1, nil
		}

		currIdx := 1
		// iterating over each element in the list

		for currIdx < len(bencodedString) && bencodedString[currIdx] != 'e' {
			// decode key
			if !unicode.IsDigit(rune(bencodedString[currIdx])) {
				// key is not a string
				return "", -1, fmt.Errorf("error occured decoding: %c at: %d, dict key is not a string", bencodedString[0], 0)
			}

			decodedKey, relativeEndIdx, decodingErr := decodeBencodeHelper(bencodedString[currIdx:])
			if decodingErr != nil {
				return "", -1, fmt.Errorf("error occured decoding: %c at: %d, %w", bencodedString[currIdx], currIdx, decodingErr)
			}

			decodedKeyStr, ok := decodedKey.(string)
			if !ok {
				return "", -1, fmt.Errorf("assert failed for: %c at: %d, not a string", bencodedString[currIdx], currIdx)
			}

			// decode value
			currIdx = currIdx + relativeEndIdx + 1

			decodedValue, relativeEndIdx, decodingErr := decodeBencodeHelper(bencodedString[currIdx:])
			if decodingErr != nil {
				return "", -1, fmt.Errorf("error occured decoding: %c at: %d, %w", bencodedString[currIdx], currIdx, decodingErr)
			}

			// add key, value pair to dictionary
			dict[decodedKeyStr] = decodedValue

			currIdx = currIdx + relativeEndIdx + 1
		}

		return dict, currIdx, nil
	} else {
		return "", -1, fmt.Errorf("error occured decoding: %c at: %d", bencodedString[0], 0)
	}
}

// Example:
//   - 5:hello -> hello
//   - 10:hello12345 -> hello12345
func DecodeBencode(bencodedString string) (interface{}, error) {
	decodedString, _, err := decodeBencodeHelper(bencodedString)
	return decodedString, err
}
