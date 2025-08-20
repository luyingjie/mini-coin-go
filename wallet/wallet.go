package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"log"
	"math/big"

	"golang.org/x/crypto/ripemd160"
)

const ( // 版本和地址校验和长度
	version            = byte(0x00)
	addressChecksumLen = 4
)

// Wallet 存储私钥和公钥
type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

// NewKeyPair 创建一个新的密钥对
func NewKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return *private, pubKey
}

// NewWallet 创建并返回一个新的钱包
func NewWallet() *Wallet {
	private, public := NewKeyPair()
	wallet := Wallet{private, public}

	return &wallet
}

// GetAddress 返回钱包地址
func (w Wallet) GetAddress() []byte {
	pubKeyHash := HashPubKey(w.PublicKey)

	versionedPayload := append([]byte{version}, pubKeyHash...)
	checksum := checksum(versionedPayload)

	fullPayload := append(versionedPayload, checksum...)
	address := Base58Encode(fullPayload)

	return address
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

// checksum 为公钥哈希生成校验和
func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])

	return secondSHA[:addressChecksumLen]
}

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
