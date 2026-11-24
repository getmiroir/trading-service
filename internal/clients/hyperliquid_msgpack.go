package clients

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

// hlOrderedMap is a slice of [key, value] pairs that encodes as a msgpack map while
// preserving insertion order. Hyperliquid's L1 action hash requires msgpack encoding
// with keys in the exact same order as the Python SDK produces them.
type hlOrderedMap [][2]interface{}

// MarshalJSON encodes hlOrderedMap as a JSON object (not array),
// preserving key insertion order for readability.
func (m hlOrderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, kv := range m {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyJSON, err := json.Marshal(fmt.Sprintf("%v", kv[0]))
		if err != nil {
			return nil, err
		}
		buf.Write(keyJSON)
		buf.WriteByte(':')
		valJSON, err := json.Marshal(kv[1])
		if err != nil {
			return nil, err
		}
		buf.Write(valJSON)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// hlMsgpackEncode encodes v as msgpack bytes.
// Supports: nil, bool, int, int64, uint64, string, []interface{}, hlOrderedMap.
func hlMsgpackEncode(v interface{}) ([]byte, error) {
	var buf []byte
	if err := hlMsgpackAppend(&buf, v); err != nil {
		return nil, err
	}
	return buf, nil
}

func hlMsgpackAppend(buf *[]byte, v interface{}) error {
	switch val := v.(type) {
	case nil:
		*buf = append(*buf, 0xc0)
	case bool:
		if val {
			*buf = append(*buf, 0xc3)
		} else {
			*buf = append(*buf, 0xc2)
		}
	case int:
		hlMsgpackAppendInt(buf, int64(val))
	case int64:
		hlMsgpackAppendInt(buf, val)
	case uint64:
		hlMsgpackAppendUint(buf, val)
	case string:
		hlMsgpackAppendStr(buf, val)
	case []interface{}:
		n := len(val)
		if n <= 15 {
			*buf = append(*buf, byte(0x90|n))
		} else if n <= 65535 {
			b := make([]byte, 3)
			b[0] = 0xdc
			binary.BigEndian.PutUint16(b[1:], uint16(n))
			*buf = append(*buf, b...)
		} else {
			return fmt.Errorf("hlMsgpackEncode: array too large: %d", n)
		}
		for _, elem := range val {
			if err := hlMsgpackAppend(buf, elem); err != nil {
				return err
			}
		}
	case hlOrderedMap:
		n := len(val)
		if n <= 15 {
			*buf = append(*buf, byte(0x80|n))
		} else if n <= 65535 {
			b := make([]byte, 3)
			b[0] = 0xde
			binary.BigEndian.PutUint16(b[1:], uint16(n))
			*buf = append(*buf, b...)
		} else {
			return fmt.Errorf("hlMsgpackEncode: map too large: %d", n)
		}
		for _, kv := range val {
			if err := hlMsgpackAppend(buf, kv[0]); err != nil {
				return err
			}
			if err := hlMsgpackAppend(buf, kv[1]); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("hlMsgpackEncode: unsupported type %T", v)
	}
	return nil
}

func hlMsgpackAppendInt(buf *[]byte, v int64) {
	switch {
	case v >= 0:
		hlMsgpackAppendUint(buf, uint64(v))
	case v >= -32:
		// negative fixint: 0xe0–0xff
		*buf = append(*buf, byte(v))
	case v >= -128:
		*buf = append(*buf, 0xd0, byte(int8(v)))
	case v >= -32768:
		b := make([]byte, 3)
		b[0] = 0xd1
		binary.BigEndian.PutUint16(b[1:], uint16(int16(v)))
		*buf = append(*buf, b...)
	case v >= -2147483648:
		b := make([]byte, 5)
		b[0] = 0xd2
		binary.BigEndian.PutUint32(b[1:], uint32(int32(v)))
		*buf = append(*buf, b...)
	default:
		b := make([]byte, 9)
		b[0] = 0xd3
		binary.BigEndian.PutUint64(b[1:], uint64(v))
		*buf = append(*buf, b...)
	}
}

func hlMsgpackAppendUint(buf *[]byte, v uint64) {
	switch {
	case v <= 127:
		// positive fixint
		*buf = append(*buf, byte(v))
	case v <= 255:
		*buf = append(*buf, 0xcc, byte(v))
	case v <= 65535:
		*buf = append(*buf, 0xcd, byte(v>>8), byte(v))
	case v <= 4294967295:
		b := make([]byte, 5)
		b[0] = 0xce
		binary.BigEndian.PutUint32(b[1:], uint32(v))
		*buf = append(*buf, b...)
	default:
		b := make([]byte, 9)
		b[0] = 0xcf
		binary.BigEndian.PutUint64(b[1:], v)
		*buf = append(*buf, b...)
	}
}

func hlMsgpackAppendStr(buf *[]byte, s string) {
	b := []byte(s)
	n := len(b)
	switch {
	case n <= 31:
		// fixstr: 0xa0–0xbf
		*buf = append(*buf, byte(0xa0|n))
	case n <= 255:
		// str8
		*buf = append(*buf, 0xd9, byte(n))
	case n <= 65535:
		// str16
		*buf = append(*buf, 0xda, byte(n>>8), byte(n))
	default:
		panic("hlMsgpackAppendStr: string too large")
	}
	*buf = append(*buf, b...)
}
