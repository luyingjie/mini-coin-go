package wallet

import (
	"os"
	"testing"
)

// TestNewWallets 测试创建钱包集合
func TestNewWallets(t *testing.T) {
	setupTestEnvironment()
	defer teardownTestEnvironment()

	wallets, err := NewWallets()
	if err != nil {
		t.Errorf("Failed to create wallets: %v", err)
	}

	if wallets == nil {
		t.Error("Wallets should not be nil")
	}

	if wallets.Wallets == nil {
		t.Error("Wallets map should not be nil")
	}
}

// TestWallets_CreateWallet 测试创建钱包
func TestWallets_CreateWallet(t *testing.T) {
	setupTestEnvironment()
	defer teardownTestEnvironment()

	wallets, _ := NewWallets()

	address := wallets.CreateWallet()

	if address == "" {
		t.Error("Address should not be empty")
	}

	// 验证钱包是否被正确存储
	if _, exists := wallets.Wallets[address]; !exists {
		t.Error("Wallet should be stored in the collection")
	}

	// 测试创建多个钱包
	address2 := wallets.CreateWallet()
	if address == address2 {
		t.Error("Different wallets should have different addresses")
	}

	if len(wallets.Wallets) != 2 {
		t.Errorf("Expected 2 wallets, got %d", len(wallets.Wallets))
	}
}

// TestWallets_GetAddresses 测试获取地址列表
func TestWallets_GetAddresses(t *testing.T) {
	setupTestEnvironment()
	defer teardownTestEnvironment()

	wallets, _ := NewWallets()

	// 初始应该没有地址
	addresses := wallets.GetAddresses()
	if len(addresses) != 0 {
		t.Errorf("Expected 0 addresses, got %d", len(addresses))
	}

	// 创建钱包后应该有地址
	addr1 := wallets.CreateWallet()
	addr2 := wallets.CreateWallet()

	addresses = wallets.GetAddresses()
	if len(addresses) != 2 {
		t.Errorf("Expected 2 addresses, got %d", len(addresses))
	}

	// 检查地址是否正确
	found1, found2 := false, false
	for _, addr := range addresses {
		if addr == addr1 {
			found1 = true
		}
		if addr == addr2 {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Error("All created addresses should be in the list")
	}
}

// TestWallets_GetWallet 测试获取特定钱包
func TestWallets_GetWallet(t *testing.T) {
	setupTestEnvironment()
	defer teardownTestEnvironment()

	wallets, _ := NewWallets()
	address := wallets.CreateWallet()

	wallet := wallets.GetWallet(address)

	if len(wallet.PrivKey) == 0 {
		t.Error("Retrieved wallet should have a private key")
	}

	if len(wallet.PubKey) == 0 {
		t.Error("Retrieved wallet should have a public key")
	}

	// 验证地址匹配
	retrievedAddress := wallet.GetAddress()
	if string(retrievedAddress) != address {
		t.Error("Retrieved wallet should generate the same address")
	}
}

// TestWallets_SaveToFile_LoadFromFile 测试文件保存和加载
func TestWallets_SaveToFile_LoadFromFile(t *testing.T) {
	setupTestEnvironment()
	defer teardownTestEnvironment()

	// 创建钱包并保存
	wallets1, _ := NewWallets()
	addr1 := wallets1.CreateWallet()
	addr2 := wallets1.CreateWallet()

	wallets1.SaveToFile()

	// 验证文件是否存在
	if _, err := os.Stat("wallet.dat"); os.IsNotExist(err) {
		t.Error("Wallet file should exist after saving")
	}

	// 创建新的钱包集合并从文件加载
	wallets2, err := NewWallets()
	if err != nil {
		t.Errorf("Failed to load wallets from file: %v", err)
	}

	// 验证加载的钱包数量
	if len(wallets2.Wallets) != 2 {
		t.Errorf("Expected 2 loaded wallets, got %d", len(wallets2.Wallets))
	}

	// 验证地址是否相同
	addresses := wallets2.GetAddresses()
	found1, found2 := false, false
	for _, addr := range addresses {
		if addr == addr1 {
			found1 = true
		}
		if addr == addr2 {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Error("All saved addresses should be loaded")
	}
}
