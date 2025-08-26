package blockchain

import (
	"testing"
)

// TestBase58EncodeDecode 测试 Base58 编码和解码
func TestBase58EncodeDecode(t *testing.T) {
	testData := [][]byte{
		[]byte("hello world"),
		[]byte(""),
		[]byte{0, 1, 2, 3, 4},
		[]byte{0, 0, 1, 2, 3},
	}

	for _, data := range testData {
		encoded := Base58Encode(data)
		decoded := Base58Decode(encoded)

		if string(data) != string(decoded) {
			t.Errorf("Base58 encode/decode failed for %v. Original: %v, Decoded: %v", data, data, decoded)
		}
	}
}

// TestValidateAddress 测试地址验证
func TestValidateAddress(t *testing.T) {
	// 测试有效地址（使用真实创建的地址）
	validAddresses := []string{
		"17BJsKiaasXt4S7EKe9PhtZAZF74VJZwGv", // 真实创建的地址
	}

	for _, addr := range validAddresses {
		if !ValidateAddress(addr) {
			t.Errorf("Address %s should be valid", addr)
		}
	}

	// 测试无效地址
	invalidAddresses := []string{
		"",
		"invalid",
		"123",
		"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfN", // 太短
	}

	for _, addr := range invalidAddresses {
		if ValidateAddress(addr) {
			t.Errorf("Address %s should be invalid", addr)
		}
	}
}

// TestHashPubKey 测试公钥哈希
func TestHashPubKey(t *testing.T) {
	pubKey := []byte("test_public_key")

	hash1 := HashPubKey(pubKey)
	hash2 := HashPubKey(pubKey)

	if len(hash1) == 0 {
		t.Error("Hash should not be empty")
	}

	// 相同的输入应该产生相同的哈希
	if string(hash1) != string(hash2) {
		t.Error("Same input should produce same hash")
	}

	// 不同的输入应该产生不同的哈希
	differentPubKey := []byte("different_public_key")
	hash3 := HashPubKey(differentPubKey)

	if string(hash1) == string(hash3) {
		t.Error("Different inputs should produce different hashes")
	}
}

// TestChecksum 测试校验和生成
func TestChecksum(t *testing.T) {
	payload := []byte("test_payload")

	checksum1 := checksum(payload)
	checksum2 := checksum(payload)

	if len(checksum1) != addressChecksumLen {
		t.Errorf("Expected checksum length %d, got %d", addressChecksumLen, len(checksum1))
	}

	// 相同的输入应该产生相同的校验和
	if string(checksum1) != string(checksum2) {
		t.Error("Same input should produce same checksum")
	}
}

// TestIntToHex 测试整数到十六进制转换
func TestIntToHex(t *testing.T) {
	testCases := []int64{0, 1, 255, 65535, -1}

	for _, num := range testCases {
		hex := IntToHex(num)

		if len(hex) == 0 {
			t.Errorf("Hex representation should not be empty for %d", num)
		}

		// 测试相同数字产生相同结果
		hex2 := IntToHex(num)
		if string(hex) != string(hex2) {
			t.Errorf("Same number should produce same hex representation")
		}
	}
}
