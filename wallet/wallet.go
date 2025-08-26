package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"log"

	"mini-coin-go/blockchain"

	"golang.org/x/crypto/ripemd160"
)

const ( // 版本和地址校验和长度
	version            = byte(0x00)
	addressChecksumLen = 4
)

// Wallet 存储私钥和公钥
type Wallet struct {
	PrivKey []byte
	PubKey  []byte
}

// NewKeyPair 创建一个新的密钥对
func NewKeyPair() ([]byte, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	privKey := private.D.Bytes()

	return privKey, pubKey
}

// NewWallet 创建并返回一个新的钱包
func NewWallet() *Wallet {
	privKey, pubKey := NewKeyPair()
	wallet := Wallet{privKey, pubKey}

	return &wallet
}

// GetAddress 返回钱包地址
func (w Wallet) GetAddress() []byte {
	pubKeyHash := HashPubKey(w.PubKey)

	versionedPayload := append([]byte{version}, pubKeyHash...)
	checksum := checksum(versionedPayload)

	fullPayload := append(versionedPayload, checksum...)
	address := blockchain.Base58Encode(fullPayload)

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
