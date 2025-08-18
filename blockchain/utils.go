package blockchain

import (
	"bytes"
	"encoding/binary"
	"log"
)

// IntToHex 用于将 int64 转换为字节数组
func IntToHex(n int64) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, n)
	if err != nil {
		log.Panic(err)
	}
	return buf.Bytes()
}
