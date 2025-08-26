package blockchain

import (
	"os"
	"testing"
)

const (
	testDBFile   = "blockchain_test.db"
	backupDBFile = "blockchain.db.bak"
)

// setupTestEnvironment 设置测试环境，备份现有数据文件
func setupTestEnvironment() {
	if _, err := os.Stat("blockchain.db"); err == nil {
		os.Rename("blockchain.db", backupDBFile)
	}
}

// teardownTestEnvironment 清理测试环境，恢复数据文件
func teardownTestEnvironment() {
	os.Remove("blockchain.db")

	if _, err := os.Stat(backupDBFile); err == nil {
		os.Rename(backupDBFile, "blockchain.db")
	}
}

// TestNewBlockchain 测试创建新区块链
func TestNewBlockchain(t *testing.T) {
	setupTestEnvironment()
	defer teardownTestEnvironment()

	address := "17BJsKiaasXt4S7EKe9PhtZAZF74VJZwGv"
	bc := NewBlockchain(address)
	defer bc.DB.Close()

	if bc == nil {
		t.Error("Failed to create blockchain")
	}

	if bc.tip == nil {
		t.Error("Blockchain tip should not be nil")
	}
}

// TestBlockchain_MineBlock 测试挖矿功能
func TestBlockchain_MineBlock(t *testing.T) {
	setupTestEnvironment()
	defer teardownTestEnvironment()

	address := "17BJsKiaasXt4S7EKe9PhtZAZF74VJZwGv"
	bc := NewBlockchain(address)
	defer bc.DB.Close()

	// 创建一个新交易
	tx := NewCoinbaseTX(address, "")

	// 挖矿
	bc.MineBlock([]*Transaction{tx}, address)

	// 验证区块链长度
	iterator := bc.Iterator()
	count := 0
	for {
		block := iterator.Next()
		count++

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	if count < 2 { // 至少应该有创世区块 + 新挖的区块
		t.Errorf("Expected at least 2 blocks, got %d", count)
	}
}

// TestUTXOSet_Reindex 测试 UTXO 重建索引
func TestUTXOSet_Reindex(t *testing.T) {
	setupTestEnvironment()
	defer teardownTestEnvironment()

	address := "17BJsKiaasXt4S7EKe9PhtZAZF74VJZwGv"
	bc := NewBlockchain(address)
	defer bc.DB.Close()

	utxoSet := UTXOSet{bc}
	utxoSet.Reindex()

	// 查找 UTXO
	pubKeyHash := Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	utxos := utxoSet.FindUTXO(pubKeyHash)

	if len(utxos) == 0 {
		t.Error("Expected at least one UTXO for the address")
	}
}
