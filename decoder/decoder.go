package decoder

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

// // helper, to be removed later
// func PeekBytes(reader *bufio.Reader, msg string) {
// 	fmt.Println("peeking at:", msg)
// 	p, err := reader.Peek(20)
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	log.Println("peeking 20 bytes", string(p))
// }

func decodeBencodeHelper(reader *bufio.Reader) (interface{}, error) {
	ch, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	switch ch {
	case 'i':
		// integer
		intBuffer, err := readBytesUntil(reader, 'e')
		if err != nil {
			return nil, err
		}
		// 'e' has also been read into the buffer, so remove it
		intBuffer = intBuffer[:len(intBuffer)-1]

		// parse bytes to int64 as per torrent specs
		integer, err := strconv.ParseInt(string(intBuffer), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing to int64 %v", err)
		}

		return integer, nil

	case 'l':
		// list
		var list []interface{}
		for {
			c, err := reader.ReadByte()
			if err == nil {
				if c == 'e' {
					// return result as end of list is reached
					return list, nil
				}

				// list not ended
				reader.UnreadByte()
			}

			item, err := decodeBencodeHelper(reader)
			if err != nil {
				return nil, err
			}

			list = append(list, item)
		}
	case 'd':
		// dictionary
		dict := map[string]interface{}{}

		// read key, value pairs iteratively
		for {
			c, err := reader.ReadByte()
			if err == nil {
				if c == 'e' {
					// return result, as end of the dictionary reached
					return dict, nil
				}

				// dictionary not ended
				reader.UnreadByte()
			}

			// read key
			keyI, err := decodeBencodeHelper(reader)
			if err != nil {
				return nil, err
			}

			// As per specifiction the keys of the dictionary must be string
			key, ok := keyI.(string)
			if !ok {
				return nil, fmt.Errorf("error: dictionary key is not a string, got: %v", keyI)
			}

			// read value
			value, err := decodeBencodeHelper(reader)
			if err != nil {
				return nil, fmt.Errorf("error in parsing dictionary, failed to get value of key %s", key)
			}

			dict[key] = value
		}
	default:
		// string

		// unread the first byte which is part of string length
		err := reader.UnreadByte()
		if err != nil {
			log.Println("error unreading the bytes", err)
		}

		stringLenBuffer, err := readBytesUntil(reader, ':')
		if err != nil {
			return nil, err
		}

		// ':' is also read into buffer, so remove it
		stringLenBuffer = stringLenBuffer[:len(stringLenBuffer)-1]

		// parse string length to int64
		stringLen, err := strconv.ParseInt(string(stringLenBuffer), 10, 64)
		if err != nil {
			return nil, err
		}

		stringBuf := make([]byte, stringLen)

		_, err = readAtLeast(reader, stringBuf, int(stringLen))

		return string(stringBuf), err
	}
}

// readBytesUntil reads and copies the byte into a buffer (new byte slice) until
// byte 'delim' is encountered. The copied bytes are returned as the result
func readBytesUntil(reader *bufio.Reader, delim byte) ([]byte, error) {
	remBytes := reader.Buffered()
	var buffer []byte
	var err error

	// using Peek to directly access the buffered data without copying it into buffer
	if buffer, err = reader.Peek(remBytes); err != nil {
		return nil, err
	}

	// check if the delimiter can be found in the bytes buffer
	if i := bytes.IndexByte(buffer, delim); i >= 0 {
		return reader.ReadSlice(delim)
	}

	return reader.ReadBytes(delim)
}

// readAtLeast emulates the behaviour of io.ReadAtLeast for bufio.Reader
// It reads from reader into buf at least min bytes
// It returns the number of bytes copied and an error if fewer bytes were read.
func readAtLeast(reader *bufio.Reader, buf []byte, min int) (n int, err error) {
	// validate the pre-allocated buffer
	if len(buf) < min {
		return 0, io.ErrShortBuffer
	}

	// reading data iteratively assuming that data will not be read
	// form buffer in a sing .Read() call
	// n denotes the last position of copied bzytes into buf
	for n < min && err == nil {
		var nn int
		// read and append to buffer
		nn, err = reader.Read(buf[n:])
		n += nn
	}

	if n >= min {
		err = nil
	} else if n > 0 && err == io.EOF {
		err = io.ErrUnexpectedEOF
	}

	return
}

func DecodeBencode(bencoded []byte) (interface{}, error) {
	byteReader := bytes.NewReader(bencoded)
	bufReader := bufio.NewReader(byteReader)
	return decodeBencodeHelper(bufReader)
}
