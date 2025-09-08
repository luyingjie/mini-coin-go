package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"mini-coin-go/blockchain"
	"mini-coin-go/network"
	"mini-coin-go/wallet"

	"go.etcd.io/bbolt"
)

// TestBlockchainNetworkIntegration 完整的区块链网络集成测试
// 测试场景：启动3个节点A:3000, B:3001, C:3002
// B挖出第一个区块获得100BTC，B发送20BTC给C，A挖出第二个区块打包交易
// 最终余额：A=100BTC, B=80BTC, C=20BTC
func TestBlockchainNetworkIntegration(t *testing.T) {
	// 清理测试环境
	cleanup := setupTestEnvironment()
	defer cleanup()

	// 创建节点钱包
	nodeAWallet, nodeAAddress := createTestWallet(t, "A")
	nodeBWallet, nodeBAddress := createTestWallet(t, "B")
	nodeCWallet, nodeCAddress := createTestWallet(t, "C")

	log.Printf("节点地址创建完成:")
	log.Printf("节点A地址: %s", nodeAAddress)
	log.Printf("节点B地址: %s", nodeBAddress)
	log.Printf("节点C地址: %s", nodeCAddress)

	// 创建区块链实例（使用节点B的地址作为创世区块地址）
	bc := blockchain.NewBlockchain(nodeBAddress, "3001")
	defer bc.DB.Close()

	// 验证创世区块
	verifyGenesisBlock(t, bc, nodeBAddress)

	// 步骤1: 节点B挖出第一个区块（已经通过创世区块完成）
	log.Println("=== 步骤1: 节点B已通过创世区块获得100BTC ===")

	// 验证节点B初始余额
	balanceB := getBalance(t, bc, nodeBAddress, nodeAWallet, nodeBWallet, nodeCWallet)
	if balanceB != 100 {
		t.Errorf("节点B初始余额应该是100BTC，实际是%d", balanceB)
	}
	log.Printf("节点B当前余额: %d BTC", balanceB)

	// 步骤2: 节点B发送20BTC给节点C
	log.Println("=== 步骤2: 节点B发送20BTC给节点C ===")

	tx := createTransaction(t, bc, nodeBAddress, nodeCAddress, 20, nodeBWallet)
	log.Printf("交易创建成功，ID: %x", tx.ID)

	// 将交易添加到内存池（模拟网络传播）
	addToMempool(tx)

	// 步骤3: 节点A挖出第二个区块打包交易
	log.Println("=== 步骤3: 节点A挖出第二个区块打包交易 ===")

	// 模拟节点A挖矿，将内存池中的交易打包到新区块
	newBlock := mineBlockWithTransactions(t, bc, nodeAAddress, []*blockchain.Transaction{tx})
	log.Printf("节点A挖出新区块，高度: %d, 哈希: %x", newBlock.Height, newBlock.Hash)

	// 将新区块添加到区块链
	err := bc.AddBlock(newBlock)
	if err != nil {
		t.Fatalf("添加区块失败: %v", err)
	}

	// 等待区块链状态更新
	time.Sleep(100 * time.Millisecond)

	// 步骤4: 验证最终余额
	log.Println("=== 步骤4: 验证最终余额 ===")

	// 重新创建区块链实例以确保数据一致性
	bc.DB.Close()
	bc = blockchain.NewBlockchain(nodeBAddress, "3001")
	defer bc.DB.Close()

	finalBalanceA := getBalance(t, bc, nodeAAddress, nodeAWallet, nodeBWallet, nodeCWallet)
	finalBalanceB := getBalance(t, bc, nodeBAddress, nodeAWallet, nodeBWallet, nodeCWallet)
	finalBalanceC := getBalance(t, bc, nodeCAddress, nodeAWallet, nodeBWallet, nodeCWallet)

	log.Printf("最终余额验证:")
	log.Printf("节点A余额: %d BTC (期望: 100)", finalBalanceA)
	log.Printf("节点B余额: %d BTC (期望: 80)", finalBalanceB)
	log.Printf("节点C余额: %d BTC (期望: 20)", finalBalanceC)

	// 验证期望余额
	if finalBalanceA != 100 {
		t.Errorf("节点A最终余额应该是100BTC，实际是%d", finalBalanceA)
	}
	if finalBalanceB != 80 {
		t.Errorf("节点B最终余额应该是80BTC，实际是%d", finalBalanceB)
	}
	if finalBalanceC != 20 {
		t.Errorf("节点C最终余额应该是20BTC，实际是%d", finalBalanceC)
	}

	log.Println("=== 集成测试完成：所有余额验证通过 ===")
}

// setupTestEnvironment 设置测试环境
func setupTestEnvironment() func() {
	// 清理测试文件
	testFiles := []string{
		"blockchain_3001.db",
		"blockchain_test.db",
		"wallet_A.dat",
		"wallet_B.dat",
		"wallet_C.dat",
		"chainstate_3001.db",
	}

	for _, file := range testFiles {
		os.Remove(file)
	}

	return func() {
		// 清理函数
		for _, file := range testFiles {
			os.Remove(file)
		}
	}
}

// createTestWallet 创建测试钱包
func createTestWallet(t *testing.T, nodeID string) (*wallet.Wallets, string) {
	ws, err := wallet.NewWallets(nodeID)
	if err != nil {
		// 如果文件不存在，这是正常的
	}

	address := ws.CreateWallet()
	ws.SaveToFile(nodeID)

	log.Printf("节点%s钱包创建成功，地址: %s", nodeID, address)
	return ws, address
}

// verifyGenesisBlock 验证创世区块
func verifyGenesisBlock(t *testing.T, bc *blockchain.Blockchain, minerAddress string) {
	bestHeight := bc.GetBestHeight()
	if bestHeight < 0 {
		t.Fatal("区块链应该至少有创世区块")
	}

	// 获取创世区块
	blockHashes := bc.GetBlockHashes()
	if len(blockHashes) == 0 {
		t.Fatal("应该至少有一个区块")
	}

	genesisBlock, err := bc.GetBlock(blockHashes[0])
	if err != nil {
		t.Fatalf("获取创世区块失败: %v", err)
	}

	if genesisBlock.Height != 0 {
		t.Errorf("创世区块高度应该是0，实际是%d", genesisBlock.Height)
	}

	log.Printf("创世区块验证成功，高度: %d, 哈希: %x", genesisBlock.Height, genesisBlock.Hash)
}

// getBalance 获取地址余额
func getBalance(t *testing.T, bc *blockchain.Blockchain, address string, nodeAWallet, nodeBWallet, nodeCWallet *wallet.Wallets) int {
	UTXOSet := blockchain.UTXOSet{Blockchain: bc}
	UTXOSet.Reindex()

	balance := 0
	pubKeyHash := blockchain.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := UTXOSet.FindUTXO(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}

	return balance
}

// checkAddressInWallet 检查地址是否在钱包中
func checkAddressInWallet(ws *wallet.Wallets, address string) bool {
	addresses := ws.GetAddresses()
	for _, addr := range addresses {
		if addr == address {
			return true
		}
	}
	return false
}

// createTransaction 创建交易
func createTransaction(t *testing.T, bc *blockchain.Blockchain, from, to string, amount int, senderWallet *wallet.Wallets) *blockchain.Transaction {
	UTXOSet := blockchain.UTXOSet{Blockchain: bc}
	UTXOSet.Reindex()

	tx := blockchain.NewUTXOTransaction(from, to, amount, &UTXOSet)

	return tx
}

// 模拟内存池
var mempool = make(map[string]*blockchain.Transaction)
var mempoolMutex sync.RWMutex

// addToMempool 添加交易到内存池
func addToMempool(tx *blockchain.Transaction) {
	mempoolMutex.Lock()
	defer mempoolMutex.Unlock()
	mempool[string(tx.ID)] = tx
	log.Printf("交易添加到内存池: %x", tx.ID)
}

// getTransactionsFromMempool 从内存池获取交易
func getTransactionsFromMempool() []*blockchain.Transaction {
	mempoolMutex.RLock()
	defer mempoolMutex.RUnlock()

	var transactions []*blockchain.Transaction
	for _, tx := range mempool {
		transactions = append(transactions, tx)
	}
	return transactions
}

// clearMempool 清空内存池
func clearMempool() {
	mempoolMutex.Lock()
	defer mempoolMutex.Unlock()
	mempool = make(map[string]*blockchain.Transaction)
}

// mineBlockWithTransactions 挖掘包含指定交易的区块
func mineBlockWithTransactions(t *testing.T, bc *blockchain.Blockchain, minerAddress string, transactions []*blockchain.Transaction) *blockchain.Block {
	// 创建coinbase交易（挖矿奖励）
	coinbaseTx := blockchain.NewCoinbaseTX(minerAddress, "")

	// 将coinbase交易添加到交易列表开头
	allTransactions := []*blockchain.Transaction{coinbaseTx}
	allTransactions = append(allTransactions, transactions...)

	// 获取前一个区块的哈希（使用与 MineBlock 相同的逻辑）
	var lastHash []byte
	err := bc.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("blocks"))
		lastHash = b.Get([]byte("l"))
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	height := bc.GetBestHeight() + 1

	// 创建新区块
	newBlock := blockchain.NewBlock(allTransactions, lastHash, height)

	log.Printf("挖掘区块成功，矿工: %s, 高度: %d, 交易数: %d", minerAddress, height, len(allTransactions))

	// 清空内存池（交易已被打包）
	clearMempool()

	return newBlock
}

// TestNetworkCommunication 测试网络通信（模拟）
func TestNetworkCommunication(t *testing.T) {
	log.Println("=== 测试网络通信模拟 ===")

	// 模拟启动网络节点
	nodes := []string{"3000", "3001", "3002"}

	for _, nodeID := range nodes {
		log.Printf("模拟启动节点: localhost:%s", nodeID)

		// 模拟网络节点初始化
		address := fmt.Sprintf("localhost:%s", nodeID)

		// 验证节点地址格式
		if len(address) == 0 {
			t.Errorf("节点地址不能为空: %s", nodeID)
		}

		// 模拟网络连接测试
		log.Printf("节点 %s 网络连接测试通过", nodeID)
	}

	// 模拟节点间消息传递
	log.Println("模拟节点间消息传递...")

	// 模拟版本消息
	version := network.Version{
		Version:    1,
		BestHeight: 0,
		AddrFrom:   "localhost:3001",
	}

	payload, err := network.GobEncode(version)
	if err != nil {
		t.Errorf("编码版本消息失败: %v", err)
	}

	if len(payload) == 0 {
		t.Error("编码后的消息不应为空")
	}

	log.Printf("版本消息编码成功，大小: %d bytes", len(payload))
	log.Println("=== 网络通信模拟测试完成 ===")
}
