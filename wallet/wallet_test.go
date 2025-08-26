package wallet

import (
	"os"
	"testing"

	"mini-coin-go/blockchain"
)

const (
	testWalletFile   = "wallet_test.dat"
	backupWalletFile = "wallet.dat.bak"
)

// setupTestEnvironment 设置测试环境，备份现有钱包文件
func setupTestEnvironment() {
	if _, err := os.Stat("wallet.dat"); err == nil {
		os.Rename("wallet.dat", backupWalletFile)
	}
}

// teardownTestEnvironment 清理测试环境，恢复钱包文件
func teardownTestEnvironment() {
	os.Remove("wallet.dat")

	if _, err := os.Stat(backupWalletFile); err == nil {
		os.Rename(backupWalletFile, "wallet.dat")
	}
}

// TestNewWallet 测试创建新钱包
func TestNewWallet(t *testing.T) {
	wallet := NewWallet()

	if wallet == nil {
		t.Error("Failed to create wallet")
	}

	if len(wallet.PrivKey) == 0 {
		t.Error("Private key should not be empty")
	}

	if len(wallet.PubKey) == 0 {
		t.Error("Public key should not be empty")
	}

	// 测试地址生成
	address := wallet.GetAddress()
	if len(address) == 0 {
		t.Error("Address should not be empty")
	}

	// 验证地址格式
	if !blockchain.ValidateAddress(string(address)) {
		t.Error("Generated address should be valid")
	}
}

// TestNewKeyPair 测试密钥对生成
func TestNewKeyPair(t *testing.T) {
	privKey1, pubKey1 := NewKeyPair()
	privKey2, pubKey2 := NewKeyPair()

	if len(privKey1) == 0 || len(pubKey1) == 0 {
		t.Error("Key pair should not be empty")
	}

	// 不同的调用应该生成不同的密钥对
	if string(privKey1) == string(privKey2) {
		t.Error("Different calls should generate different private keys")
	}

	if string(pubKey1) == string(pubKey2) {
		t.Error("Different calls should generate different public keys")
	}
}

// TestWallet_GetAddress 测试地址生成
func TestWallet_GetAddress(t *testing.T) {
	wallet := NewWallet()

	address1 := wallet.GetAddress()
	address2 := wallet.GetAddress()

	// 同一个钱包应该生成相同的地址
	if string(address1) != string(address2) {
		t.Error("Same wallet should generate same address")
	}

	// 验证地址有效性
	if !blockchain.ValidateAddress(string(address1)) {
		t.Errorf("Generated address %s should be valid", string(address1))
	}
}

// TestHashPubKey 测试公钥哈希
func TestHashPubKey(t *testing.T) {
	wallet := NewWallet()

	hash1 := HashPubKey(wallet.PubKey)
	hash2 := HashPubKey(wallet.PubKey)

	if len(hash1) == 0 {
		t.Error("Hash should not be empty")
	}

	// 相同的公钥应该产生相同的哈希
	if string(hash1) != string(hash2) {
		t.Error("Same public key should produce same hash")
	}

	// 不同的公钥应该产生不同的哈希
	wallet2 := NewWallet()
	hash3 := HashPubKey(wallet2.PubKey)

	if string(hash1) == string(hash3) {
		t.Error("Different public keys should produce different hashes")
	}
}
