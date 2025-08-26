package blockchain

import (
	"testing"
)

// TestNewCoinbaseTX 测试 Coinbase 交易创建
func TestNewCoinbaseTX(t *testing.T) {
	address := "17BJsKiaasXt4S7EKe9PhtZAZF74VJZwGv"
	data := "Genesis Block"

	tx := NewCoinbaseTX(address, data)

	if tx == nil {
		t.Error("Failed to create coinbase transaction")
	}

	if !tx.IsCoinbase() {
		t.Error("Transaction should be a coinbase transaction")
	}

	if len(tx.Vin) != 1 {
		t.Errorf("Expected 1 input, got %d", len(tx.Vin))
	}

	if len(tx.Vout) != 1 {
		t.Errorf("Expected 1 output, got %d", len(tx.Vout))
	}

	if tx.Vout[0].Value != 100 {
		t.Errorf("Expected output value 100, got %d", tx.Vout[0].Value)
	}
}

// TestTransaction_Hash 测试交易哈希计算
func TestTransaction_Hash(t *testing.T) {
	address := "17BJsKiaasXt4S7EKe9PhtZAZF74VJZwGv"

	tx1 := NewCoinbaseTX(address, "test1")
	tx2 := NewCoinbaseTX(address, "test2")

	hash1 := tx1.Hash()
	hash2 := tx2.Hash()

	if len(hash1) == 0 || len(hash2) == 0 {
		t.Error("Transaction hash should not be empty")
	}

	// 不同的交易应该有不同的哈希
	if string(hash1) == string(hash2) {
		t.Error("Different transactions should have different hashes")
	}
}

// TestTXOutput_IsLockedWithKey 测试输出锁定检查
func TestTXOutput_IsLockedWithKey(t *testing.T) {
	address := "17BJsKiaasXt4S7EKe9PhtZAZF74VJZwGv"
	pubKeyHash := Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]

	output := TXOutput{
		Value:        50,
		ScriptPubKey: pubKeyHash,
	}

	if !output.IsLockedWithKey(pubKeyHash) {
		t.Error("Output should be locked with the provided key")
	}

	// 测试错误的密钥
	wrongKey := []byte("wrong_key")
	if output.IsLockedWithKey(wrongKey) {
		t.Error("Output should not be locked with wrong key")
	}
}

// TestTXOutputs_Serialize 测试输出序列化
func TestTXOutputs_Serialize(t *testing.T) {
	outputs := TXOutputs{
		Outputs: []TXOutput{
			{Value: 50, ScriptPubKey: []byte("key1")},
			{Value: 30, ScriptPubKey: []byte("key2")},
		},
	}

	serialized := outputs.Serialize()
	if len(serialized) == 0 {
		t.Error("Serialized data should not be empty")
	}

	// 测试反序列化
	deserialized := DeserializeOutputs(serialized)
	if len(deserialized.Outputs) != 2 {
		t.Errorf("Expected 2 outputs, got %d", len(deserialized.Outputs))
	}

	if deserialized.Outputs[0].Value != 50 {
		t.Errorf("Expected value 50, got %d", deserialized.Outputs[0].Value)
	}

	if deserialized.Outputs[1].Value != 30 {
		t.Errorf("Expected value 30, got %d", deserialized.Outputs[1].Value)
	}
}
