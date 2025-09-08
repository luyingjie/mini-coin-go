package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
)

// commandLength 命令的长度
const commandLength = 12

// CommandToBytes 将命令转换为字节数组
func CommandToBytes(command string) []byte {
	var bytes [commandLength]byte

	for i, c := range command {
		bytes[i] = byte(c)
	}

	return bytes[:]
}

// BytesToCommand 将字节数组转换为命令
func BytesToCommand(bytes []byte) string {
	var command []byte

	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}

	return fmt.Sprintf("%s", command)
}

// GobEncode 使用 gob 对数据进行编码
func GobEncode(data interface{}) ([]byte, error) {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

// ExtractCommand 从请求中提取命令
func ExtractCommand(request []byte) []byte {
	return request[:commandLength]
}

// SendMessage 统一发送消息的函数
func SendMessage(writer io.Writer, command string, payload []byte) error {
	// 构造消息，命令 + 负载
	message := append(CommandToBytes(command), payload...)
	_, err := writer.Write(message)
	return err
}
