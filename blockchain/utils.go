package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"log"
	"math/big"

	"golang.org/x/crypto/ripemd160"
)

const addressChecksumLen = 4

// Base58Encode 将字节数组编码为 Base58 格式
func Base58Encode(input []byte) []byte {
	var result []byte

	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	base := len(alphabet)

	zeros := 0
	for _, b := range input {
		if b == 0 {
			zeros++
		} else {
			break
		}
	}

	num := big.NewInt(0)
	num.SetBytes(input)

	for num.Cmp(big.NewInt(0)) > 0 {
		mod := big.NewInt(0)
		num.DivMod(num, big.NewInt(int64(base)), mod)
		result = append([]byte{alphabet[mod.Int64()]}, result...)
	}

	for i := 0; i < zeros; i++ {
		result = append([]byte{alphabet[0]}, result...)
	}

	return result
}

// Base58Decode 解码 Base58 编码的字符串
func Base58Decode(input []byte) []byte {
	result := big.NewInt(0)

	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	base := len(alphabet)

	zeros := 0
	for _, b := range input {
		if b == alphabet[0] {
			zeros++
		} else {
			break
		}
	}

	payload := input[zeros:]
	for _, b := range payload {
		charIndex := bytes.IndexByte([]byte(alphabet), b)
		result.Mul(result, big.NewInt(int64(base)))
		result.Add(result, big.NewInt(int64(charIndex)))
	}

	decoded := result.Bytes()
	decodedWithZeros := make([]byte, zeros+len(decoded))
	copy(decodedWithZeros[zeros:], decoded)

	return decodedWithZeros
}

// ValidateAddress 检查地址是否有效
func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))

	// 检查地址长度是否足够
	if len(pubKeyHash) < addressChecksumLen+1 {
		return false
	}

	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-addressChecksumLen]
	targetChecksum := checksum(append([]byte{version}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}

// checksum 为公钥哈希生成校验和
func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])

	return secondSHA[:addressChecksumLen]
}

// HashPubKey 对公钥进行哈希
func HashPubKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey)

	RIPEMD160Hasher := ripemd160.New()
	_, err := RIPEMD160Hasher.Write(publicSHA256[:])
	if err != nil {
		log.Panic(err)
	}
	publicRIPEMD160 := RIPEMD160Hasher.Sum(nil)

	return publicRIPEMD160
}

// IntToHex 用于将 int64 转换为字节数组
func IntToHex(n int64) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, n)
	if err != nil {
		log.Panic(err)
	}
	return buf.Bytes()
}
