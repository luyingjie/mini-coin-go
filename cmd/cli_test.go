package cmd

import (
	"bytes"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"mini-coin-go/blockchain"
	"mini-coin-go/wallet"
)

const (
	testDBFile       = "blockchain.db"
	testWalletFile   = "wallet.dat"
	backupDBFile     = "blockchain.db.bak"
	backupWalletFile = "wallet.dat.bak"
)

// setupTestEnvironment 设置测试环境，备份现有数据文件
func setupTestEnvironment() {
	if _, err := os.Stat(testDBFile); err == nil {
		os.Rename(testDBFile, backupDBFile)
	}
	if _, err := os.Stat(testWalletFile); err == nil {
		os.Rename(testWalletFile, backupWalletFile)
	}
}

// teardownTestEnvironment 清理测试环境，恢复数据文件
func teardownTestEnvironment() {
	os.Remove(testDBFile)
	os.Remove(testWalletFile)

	if _, err := os.Stat(backupDBFile); err == nil {
		os.Rename(backupDBFile, testDBFile)
	}
	if _, err := os.Stat(backupWalletFile); err == nil {
		os.Rename(backupWalletFile, testWalletFile)
	}
}

// captureOutput 捕获标准输出
func captureOutput(f func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var buf bytes.Buffer
	log.SetOutput(&buf) // 捕获 log 输出
	defer func() {
		os.Stdout = oldStdout
		log.SetOutput(os.Stderr)
	}()

	f()
	w.Close()
	out, _ := io.ReadAll(r)
	return buf.String() + string(out) // 合并 log 和 stdout 输出
}

// TestCLI_CreateWallet 测试创建钱包功能
func TestCLI_CreateWallet(t *testing.T) {
	setupTestEnvironment()
	defer teardownTestEnvironment()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"main", "createwallet"}

	cli := CLI{}
	output := captureOutput(func() {
		cli.Run()
	})

	if !strings.Contains(output, "Your new address:") {
		t.Errorf("Expected 'Your new address:' in output, got: %s", output)
	}

	wallets, err := wallet.NewWallets()
	if err != nil {
		t.Fatalf("Failed to load wallets: %v", err)
	}
	if len(wallets.GetAddresses()) == 0 {
		t.Error("Expected at least one wallet address, got none")
	}
}

// TestCLI_CreateBlockchain 测试创建区块链功能
func TestCLI_CreateBlockchain(t *testing.T) {
	setupTestEnvironment()
	defer teardownTestEnvironment()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// 先创建一个钱包获取地址
	os.Args = []string{"main", "createwallet"}
	cli := CLI{}
	captureOutput(func() {
		cli.Run()
	})

	wallets, _ := wallet.NewWallets()
	address := wallets.GetAddresses()[0]

	os.Args = []string{"main", "createblockchain", "-address", address}
	output := captureOutput(func() {
		cli.Run()
	})

	if !strings.Contains(output, "Done!") {
		t.Errorf("Expected 'Done!' in output, got: %s", output)
	}

	// 验证区块链是否创建成功
	bc := blockchain.NewBlockchain()
	defer bc.DB.Close()
	if bc == nil {
		t.Error("Blockchain was not created")
	}
}

// TestCLI_GetBalance 测试获取余额功能
func TestCLI_GetBalance(t *testing.T) {
	setupTestEnvironment()
	defer teardownTestEnvironment()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// 创建钱包和区块链
	os.Args = []string{"main", "createwallet"}
	cli := CLI{}
	captureOutput(func() { cli.Run() })
	wallets, _ := wallet.NewWallets()
	address := wallets.GetAddresses()[0]

	os.Args = []string{"main", "createblockchain", "-address", address}
	captureOutput(func() { cli.Run() })

	os.Args = []string{"main", "getbalance", "-address", address}
	output := captureOutput(func() {
		cli.Run()
	})

	if !strings.Contains(output, "Balance of") || !strings.Contains(output, "10") { // 假设创世区块奖励为10
		t.Errorf("Expected balance of 10 for address %s, got: %s", address, output)
	}
}

// TestCLI_Send 测试发送交易功能
func TestCLI_Send(t *testing.T) {
	setupTestEnvironment()
	defer teardownTestEnvironment()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// 创建两个钱包和区块链
	os.Args = []string{"main", "createwallet"}
	cli := CLI{}
	captureOutput(func() { cli.Run() }) // Wallet 1
	os.Args = []string{"main", "createwallet"}
	captureOutput(func() { cli.Run() }) // Wallet 2

	wallets, _ := wallet.NewWallets()
	addresses := wallets.GetAddresses()
	fromAddress := addresses[0]
	toAddress := addresses[1]

	os.Args = []string{"main", "createblockchain", "-address", fromAddress}
	captureOutput(func() { cli.Run() })

	// 发送交易
	os.Args = []string{"main", "send", "-from", fromAddress, "-to", toAddress, "-amount", "5"}
	output := captureOutput(func() {
		cli.Run()
	})

	if !strings.Contains(output, "Success!") {
		t.Errorf("Expected 'Success!' in output, got: %s", output)
	}

	// 验证余额
	os.Args = []string{"main", "getbalance", "-address", fromAddress}
	outputFrom := captureOutput(func() { cli.Run() })
	if !strings.Contains(outputFrom, "Balance of") || !strings.Contains(outputFrom, "5") { // 假设创世区块奖励为10，发送5后剩余5
		t.Errorf("Expected balance of 5 for fromAddress %s, got: %s", fromAddress, outputFrom)
	}

	os.Args = []string{"main", "getbalance", "-address", toAddress}
	outputTo := captureOutput(func() { cli.Run() })
	if !strings.Contains(outputTo, "Balance of") || !strings.Contains(outputTo, "5") {
		t.Errorf("Expected balance of 5 for toAddress %s, got: %s", toAddress, outputTo)
	}
}

// TestCLI_ListAddresses 测试列出地址功能
func TestCLI_ListAddresses(t *testing.T) {
	setupTestEnvironment()
	defer teardownTestEnvironment()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// 创建两个钱包
	os.Args = []string{"main", "createwallet"}
	cli := CLI{}
	captureOutput(func() { cli.Run() })
	os.Args = []string{"main", "createwallet"}
	captureOutput(func() { cli.Run() })

	wallets, _ := wallet.NewWallets()
	expectedAddresses := wallets.GetAddresses()

	os.Args = []string{"main", "listaddresses"}
	output := captureOutput(func() {
		cli.Run()
	})

	for _, addr := range expectedAddresses {
		if !strings.Contains(output, addr) {
			t.Errorf("Expected address %s in output, but not found. Output: %s", addr, output)
		}
	}
}

// TestCLI_PrintChain 测试打印区块链功能
func TestCLI_PrintChain(t *testing.T) {
	setupTestEnvironment()
	defer teardownTestEnvironment()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// 创建钱包和区块链
	os.Args = []string{"main", "createwallet"}
	cli := CLI{}
	captureOutput(func() { cli.Run() })
	wallets, _ := wallet.NewWallets()
	address := wallets.GetAddresses()[0]

	os.Args = []string{"main", "createblockchain", "-address", address}
	captureOutput(func() { cli.Run() })

	os.Args = []string{"main", "printchain"}
	output := captureOutput(func() {
		cli.Run()
	})

	if !strings.Contains(output, "============ Block") || !strings.Contains(output, "Prev. hash:") {
		t.Errorf("Expected blockchain output, got: %s", output)
	}
}
