package steam_appcache

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"unicode/utf16"
	// "github.com/sanity-io/litter"
)

const (
	BIN_NONE       byte = 0x00
	BIN_STRING     byte = 0x01
	BIN_INT32      byte = 0x02
	BIN_FLOAT32    byte = 0x03
	BIN_POINTER    byte = 0x04
	BIN_WIDESTRING byte = 0x05
	BIN_COLOR      byte = 0x06
	BIN_UINT64     byte = 0x07
	BIN_END        byte = 0x08
	BIN_INT64      byte = 0x0A
	BIN_END_ALT    byte = 0x0B
)

// TODO are these actually useful for anything?
type BASE_INT int64
type UINT_64 BASE_INT
type INT_64 BASE_INT
type POINTER BASE_INT
type COLOR BASE_INT

func vdfBinaryReadString(fp io.Reader, wide bool) (string, error) {
	var buf []byte
	var end = -1

	seeker, _ := fp.(io.Seeker)
	offset, _ := seeker.Seek(0, io.SeekCurrent)

	// Locate string end
	for end == -1 {
		chunk := make([]byte, 64)
		n, err := fp.Read(chunk)
		if err != nil && err != io.EOF {
			return "", err
		}

		if n == 0 && err == io.EOF {
			return "", fmt.Errorf("Unterminated cstring (offset: %d)", offset)
		}

		buf = append(buf, chunk[:n]...)
		if wide {
			end = bytes.Index(buf, []byte{0x00, 0x00})
		} else {
			end = bytes.IndexByte(buf, 0x00)
		}
	}

	if wide {
		// Ensure even boundary
		if end%2 != 0 {
			end++
		}
	}

	// Rewind file pointer
	seekOffset := int64(end - len(buf) + 1)
	if wide {
		seekOffset++
	}
	_, err := seeker.Seek(seekOffset, io.SeekCurrent)
	if err != nil {
		return "", err
	}

	// Decode string
	result := buf[:end]
	if wide {
		// Decode as UTF-16
		utf16Result := make([]uint16, len(result)/2)
		if err := binary.Read(bytes.NewReader(result), binary.LittleEndian, &utf16Result); err != nil {
			return "", err
		}
		return string(utf16.Decode(utf16Result)), nil
	} else {
		// Decode as UTF-8 or replace invalid
		decoded := string(result)
		if !isASCII(decoded) {
			return string(result), nil
		}
		return decoded, nil
	}
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

func vdfBinaryLoad(fp io.Reader, keyTable []string, mergeDuplicateKeys bool, altFormat bool) (map[string]interface{}, error) {
	stack := []map[string]interface{}{{}}
	CURRENT_BIN_END := BIN_END
	if altFormat {
		CURRENT_BIN_END = BIN_END_ALT
	}

	buf := make([]byte, 1)
	for {
		_, err := fp.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		t := buf[0]

		if t == CURRENT_BIN_END {
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
				continue
			}
			break
		}

		var key string
		if keyTable != nil {
			var index int32
			err := binary.Read(fp, binary.LittleEndian, &index)
			if err != nil {
				return nil, err
			}
			key = keyTable[index]

		} else {
			key, err = vdfBinaryReadString(fp, false)
			if err != nil {
				return nil, err
			}
		}

		current := stack[len(stack)-1]

		switch t {

		case BIN_NONE:
			if mergeDuplicateKeys {
				if _, exists := current[key]; exists {
					continue
				}
			}
			m := map[string]interface{}{}
			current[key] = m
			stack = append(stack, m)

		case BIN_STRING:
			str, err := vdfBinaryReadString(fp, false)
			if err != nil {
				return nil, err
			}
			current[key] = str

		case BIN_WIDESTRING:
			str, err := vdfBinaryReadString(fp, true)
			if err != nil {
				return nil, err
			}
			current[key] = str

		case BIN_INT32:
			var val int32
			err := binary.Read(fp, binary.LittleEndian, &val)
			if err != nil {
				return nil, err
			}
			current[key] = val

		case BIN_UINT64:
			var val uint64
			err := binary.Read(fp, binary.LittleEndian, &val)
			if err != nil {
				return nil, err
			}
			current[key] = UINT_64(val)

		case BIN_INT64:
			var val int64
			err := binary.Read(fp, binary.LittleEndian, &val)
			if err != nil {
				return nil, err
			}
			current[key] = INT_64(val)

		case BIN_POINTER:
			var val int32
			err := binary.Read(fp, binary.LittleEndian, &val)
			if err != nil {
				return nil, err
			}
			current[key] = POINTER(val)

		case BIN_COLOR:
			var val int32
			err := binary.Read(fp, binary.LittleEndian, &val)
			if err != nil {
				return nil, err
			}
			current[key] = COLOR(val)

		case BIN_FLOAT32:
			var val float32
			err := binary.Read(fp, binary.LittleEndian, &val)
			if err != nil {
				return nil, err
			}
			current[key] = val
		default:
			return nil, errors.New(fmt.Sprintf("Unknown data type at offset %d: %v", -1, t))
		}
	}

	if len(stack) != 1 {
		return nil, errors.New("Reached EOF, but Binary VDF is incomplete")
	}
	return stack[0], nil
}

func GetGameSpecificAppInfo(fp io.ReadSeeker, targetAppId uint32) (map[string]interface{}, error) {
	magic := make([]byte, 4)
	_, err := fp.Read(magic)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(magic, []byte("'DV\x07")) && !bytes.Equal(magic, []byte("(DV\x07")) && !bytes.Equal(magic, []byte(")DV\x07")) {
		return nil, errors.New(fmt.Sprintf("Invalid magic, got %v", magic))
	}

	var universe uint32
	err = binary.Read(fp, binary.LittleEndian, &universe)
	if err != nil {
		return nil, err
	}

	var keyTable []string
	if magic[0] >= 41 {
		var keyTableOffset int64
		err := binary.Read(fp, binary.LittleEndian, &keyTableOffset)
		if err != nil {
			return nil, err
		}
		offset, _ := fp.Seek(0, io.SeekCurrent)
		fp.Seek(keyTableOffset, io.SeekStart)

		var keyCount uint32
		binary.Read(fp, binary.LittleEndian, &keyCount)

		for i := 0; i < int(keyCount); i++ {
			var key string
			for {
				b := make([]byte, 1)
				_, err := fp.Read(b)
				if err != nil {
					return nil, err
				}
				if b[0] == 0 {
					break
				}
				key += string(b)
			}
			keyTable = append(keyTable, key)
		}
		fp.Seek(offset, io.SeekStart)
	}

	var apps []map[string]interface{}
	for {
		var appid uint32
		var size, infoState, lastUpdated, changeNumber uint32
		var accessToken uint64
		var sha1 [20]byte

		err = binary.Read(fp, binary.LittleEndian, &appid)
		if err != nil || appid == 0 {
			break
		}
		app := make(map[string]interface{})
		binary.Read(fp, binary.LittleEndian, &size)
		binary.Read(fp, binary.LittleEndian, &infoState)
		binary.Read(fp, binary.LittleEndian, &lastUpdated)
		binary.Read(fp, binary.LittleEndian, &accessToken)
		fp.Read(sha1[:])
		binary.Read(fp, binary.LittleEndian, &changeNumber)

		app["appid"] = appid
		app["size"] = size
		app["info_state"] = infoState
		app["last_updated"] = lastUpdated
		app["access_token"] = accessToken
		app["sha1"] = sha1
		app["change_number"] = changeNumber

		if !bytes.Equal(magic, []byte("'DV\x07")) {
			var dataSHA1 [20]byte
			fp.Read(dataSHA1[:])
			app["data_sha1"] = dataSHA1
		}

		data, err := vdfBinaryLoad(fp, keyTable, true, false)
		if err == nil {
			app["data"] = data
		} else {
			panic(err)
		}
		if appid == targetAppId {
			return app, nil
		}
		apps = append(apps, app)
		// litter.Dump(app)
	}

	// TODO should this work like upstream python version or should we just grab the app we care about and return
	// header := map[string]interface{}{
	// 	"magic":    magic,
	// 	"universe": universe,
	// }

	return nil, nil
}
