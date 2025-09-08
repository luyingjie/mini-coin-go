package sync

import (
	"fmt"
	"log"
	"sync"
	"time"

	"mini-coin-go/blockchain"
	"mini-coin-go/network/connection"
	"mini-coin-go/network/message"
)

// TransactionSyncer 交易同步器
type TransactionSyncer struct {
	blockchain   *blockchain.Blockchain
	connManager  *connection.Manager
	msgHandler   *message.Handler
	mempool      map[string]*blockchain.Transaction
	mempoolMutex sync.RWMutex
	maxPoolSize  int
	isRunning    bool
	stopCh       chan bool
	mutex        sync.RWMutex
	stats        *TxSyncStats
}

// TxSyncStats 交易同步统计信息
type TxSyncStats struct {
	TotalTxReceived  int64         // 总接收交易数
	TotalTxProcessed int64         // 总处理交易数
	TotalTxFailed    int64         // 总失败交易数
	MempoolSize      int           // 内存池大小
	AverageTxTime    time.Duration // 平均交易处理时间
	LastTxTime       time.Time     // 最后交易时间
	mutex            sync.RWMutex  // 读写锁
}

// NewTransactionSyncer 创建交易同步器
func NewTransactionSyncer(bc *blockchain.Blockchain, connManager *connection.Manager,
	msgHandler *message.Handler, maxPoolSize int) *TransactionSyncer {

	if maxPoolSize <= 0 {
		maxPoolSize = 10000
	}

	syncer := &TransactionSyncer{
		blockchain:  bc,
		connManager: connManager,
		msgHandler:  msgHandler,
		mempool:     make(map[string]*blockchain.Transaction),
		maxPoolSize: maxPoolSize,
		stopCh:      make(chan bool),
		stats:       &TxSyncStats{},
	}

	// 注册消息处理器
	syncer.registerHandlers()

	return syncer
}

// Start 启动交易同步器
func (ts *TransactionSyncer) Start() error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	if ts.isRunning {
		return fmt.Errorf("交易同步器已经在运行")
	}

	ts.isRunning = true
	log.Println("交易同步器已启动")

	// 启动清理任务
	go ts.cleanupTask()

	return nil
}

// Stop 停止交易同步器
func (ts *TransactionSyncer) Stop() error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	if !ts.isRunning {
		return nil
	}

	ts.isRunning = false
	close(ts.stopCh)

	log.Println("交易同步器已停止")
	return nil
}

// registerHandlers 注册消息处理器
func (ts *TransactionSyncer) registerHandlers() {
	ts.msgHandler.RegisterHandler("tx", ts.handleTxMessage)
	ts.msgHandler.RegisterHandler("mempool", ts.handleMempoolMessage)
}

// handleTxMessage 处理交易消息
func (ts *TransactionSyncer) handleTxMessage(msg *message.Message) error {
	start := time.Now()

	log.Printf("收到交易消息从 %s", msg.TargetAddr)

	// 反序列化交易
	tx := blockchain.DeserializeTransaction(msg.Payload)

	// 验证交易
	if !ts.validateTransaction(tx) {
		ts.updateFailedStats()
		return fmt.Errorf("交易验证失败: %x", tx.ID)
	}

	// 添加到内存池
	if err := ts.addToMempool(tx); err != nil {
		ts.updateFailedStats()
		return fmt.Errorf("添加到内存池失败: %v", err)
	}

	// 广播给其他节点
	ts.broadcastTransaction(tx, msg.TargetAddr)

	// 更新统计信息
	ts.updateProcessedStats(time.Since(start))

	log.Printf("成功处理交易: %x", tx.ID)
	return nil
}

// handleMempoolMessage 处理内存池消息
func (ts *TransactionSyncer) handleMempoolMessage(msg *message.Message) error {
	log.Printf("收到内存池请求从 %s", msg.TargetAddr)

	// 发送内存池中的交易
	return ts.sendMempoolToPeer(msg.TargetAddr)
}

// validateTransaction 验证交易
func (ts *TransactionSyncer) validateTransaction(tx *blockchain.Transaction) bool {
	// 检查交易是否已存在
	ts.mempoolMutex.RLock()
	_, exists := ts.mempool[string(tx.ID)]
	ts.mempoolMutex.RUnlock()

	if exists {
		return false // 交易已存在
	}

	// 使用区块链验证交易
	return ts.blockchain.VerifyTransaction(tx)
}

// addToMempool 添加交易到内存池
func (ts *TransactionSyncer) addToMempool(tx *blockchain.Transaction) error {
	ts.mempoolMutex.Lock()
	defer ts.mempoolMutex.Unlock()

	// 检查内存池大小限制
	if len(ts.mempool) >= ts.maxPoolSize {
		return fmt.Errorf("内存池已满")
	}

	// 检查交易是否已存在
	if _, exists := ts.mempool[string(tx.ID)]; exists {
		return fmt.Errorf("交易已存在")
	}

	ts.mempool[string(tx.ID)] = tx
	log.Printf("交易已添加到内存池: %x", tx.ID)

	return nil
}

// broadcastTransaction 广播交易给其他节点
func (ts *TransactionSyncer) broadcastTransaction(tx *blockchain.Transaction, excludeAddr string) {
	// 获取活跃的连接地址
	activeAddrs := ts.connManager.GetActiveAddresses()

	for _, addr := range activeAddrs {
		if addr == excludeAddr {
			continue // 跳过发送方
		}

		// 创建交易消息
		msg := message.NewMessage("tx", tx.Serialize(), addr)
		msg.Priority = message.PriorityNormal

		// 异步发送
		go func(targetAddr string, txMsg *message.Message) {
			if err := ts.msgHandler.Submit(txMsg); err != nil {
				log.Printf("广播交易失败到 %s: %v", targetAddr, err)
			}
		}(addr, msg)
	}
}

// sendMempoolToPeer 向节点发送内存池交易
func (ts *TransactionSyncer) sendMempoolToPeer(peerAddr string) error {
	ts.mempoolMutex.RLock()
	transactions := make([]*blockchain.Transaction, 0, len(ts.mempool))
	for _, tx := range ts.mempool {
		transactions = append(transactions, tx)
	}
	ts.mempoolMutex.RUnlock()

	// 分批发送交易（每批50个）
	batchSize := 50
	for i := 0; i < len(transactions); i += batchSize {
		end := i + batchSize
		if end > len(transactions) {
			end = len(transactions)
		}

		batch := transactions[i:end]
		if err := ts.sendTransactionBatch(batch, peerAddr); err != nil {
			log.Printf("发送交易批次失败到 %s: %v", peerAddr, err)
		}
	}

	return nil
}

// sendTransactionBatch 发送交易批次
func (ts *TransactionSyncer) sendTransactionBatch(transactions []*blockchain.Transaction, peerAddr string) error {
	for _, tx := range transactions {
		msg := message.NewMessage("tx", tx.Serialize(), peerAddr)
		msg.Priority = message.PriorityNormal

		if err := ts.msgHandler.Submit(msg); err != nil {
			return err
		}
	}

	return nil
}

// GetMempool 获取内存池交易
func (ts *TransactionSyncer) GetMempool() map[string]*blockchain.Transaction {
	ts.mempoolMutex.RLock()
	defer ts.mempoolMutex.RUnlock()

	mempool := make(map[string]*blockchain.Transaction)
	for id, tx := range ts.mempool {
		mempool[id] = tx
	}

	return mempool
}

// GetMempoolTransactions 获取内存池交易列表
func (ts *TransactionSyncer) GetMempoolTransactions() []*blockchain.Transaction {
	ts.mempoolMutex.RLock()
	defer ts.mempoolMutex.RUnlock()

	transactions := make([]*blockchain.Transaction, 0, len(ts.mempool))
	for _, tx := range ts.mempool {
		transactions = append(transactions, tx)
	}

	return transactions
}

// RemoveTransactionFromMempool 从内存池移除交易
func (ts *TransactionSyncer) RemoveTransactionFromMempool(txID []byte) {
	ts.mempoolMutex.Lock()
	defer ts.mempoolMutex.Unlock()

	delete(ts.mempool, string(txID))
}

// RemoveTransactionsFromMempool 从内存池移除多个交易
func (ts *TransactionSyncer) RemoveTransactionsFromMempool(txIDs [][]byte) {
	ts.mempoolMutex.Lock()
	defer ts.mempoolMutex.Unlock()

	for _, txID := range txIDs {
		delete(ts.mempool, string(txID))
	}
}

// cleanupTask 清理任务
func (ts *TransactionSyncer) cleanupTask() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ts.performCleanup()
		case <-ts.stopCh:
			return
		}
	}
}

// performCleanup 执行清理
func (ts *TransactionSyncer) performCleanup() {
	ts.mempoolMutex.Lock()
	defer ts.mempoolMutex.Unlock()

	// 清理过期交易（超过1小时）
	timeout := time.Hour
	var toRemove []string

	for txID, tx := range ts.mempool {
		// 这里应该检查交易的时间戳，暂时使用简单逻辑
		_ = tx
		// 如果交易创建时间超过1小时，则移除
		// 实际实现中需要在Transaction结构中添加时间戳字段
		if time.Since(time.Now()) > timeout {
			toRemove = append(toRemove, txID)
		}
	}

	for _, txID := range toRemove {
		delete(ts.mempool, txID)
	}

	if len(toRemove) > 0 {
		log.Printf("清理过期交易: %d", len(toRemove))
	}
}

// updateProcessedStats 更新处理成功统计
func (ts *TransactionSyncer) updateProcessedStats(duration time.Duration) {
	ts.stats.mutex.Lock()
	defer ts.stats.mutex.Unlock()

	ts.stats.TotalTxReceived++
	ts.stats.TotalTxProcessed++
	ts.stats.LastTxTime = time.Now()

	// 更新平均处理时间
	if ts.stats.AverageTxTime == 0 {
		ts.stats.AverageTxTime = duration
	} else {
		ts.stats.AverageTxTime = (ts.stats.AverageTxTime + duration) / 2
	}

	// 更新内存池大小
	ts.mempoolMutex.RLock()
	ts.stats.MempoolSize = len(ts.mempool)
	ts.mempoolMutex.RUnlock()
}

// updateFailedStats 更新处理失败统计
func (ts *TransactionSyncer) updateFailedStats() {
	ts.stats.mutex.Lock()
	defer ts.stats.mutex.Unlock()

	ts.stats.TotalTxReceived++
	ts.stats.TotalTxFailed++
}

// GetStats 获取交易同步统计信息
func (ts *TransactionSyncer) GetStats() *TxSyncStats {
	ts.stats.mutex.RLock()
	defer ts.stats.mutex.RUnlock()

	// 更新内存池大小
	ts.mempoolMutex.RLock()
	mempoolSize := len(ts.mempool)
	ts.mempoolMutex.RUnlock()

	return &TxSyncStats{
		TotalTxReceived:  ts.stats.TotalTxReceived,
		TotalTxProcessed: ts.stats.TotalTxProcessed,
		TotalTxFailed:    ts.stats.TotalTxFailed,
		MempoolSize:      mempoolSize,
		AverageTxTime:    ts.stats.AverageTxTime,
		LastTxTime:       ts.stats.LastTxTime,
	}
}

// GetSyncInfo 获取同步信息
func (ts *TransactionSyncer) GetSyncInfo() map[string]interface{} {
	stats := ts.GetStats()

	successRate := float64(0)
	if stats.TotalTxReceived > 0 {
		successRate = float64(stats.TotalTxProcessed) / float64(stats.TotalTxReceived) * 100
	}

	return map[string]interface{}{
		"is_running":            ts.isRunning,
		"mempool_size":          stats.MempoolSize,
		"max_pool_size":         ts.maxPoolSize,
		"total_received":        stats.TotalTxReceived,
		"total_processed":       stats.TotalTxProcessed,
		"total_failed":          stats.TotalTxFailed,
		"success_rate":          successRate,
		"average_tx_time":       stats.AverageTxTime.String(),
		"last_tx_time":          stats.LastTxTime,
		"mempool_usage_percent": float64(stats.MempoolSize) / float64(ts.maxPoolSize) * 100,
	}
}

// IsRunning 检查是否正在运行
func (ts *TransactionSyncer) IsRunning() bool {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	return ts.isRunning
}

// BroadcastTransaction 向网络广播交易
func (ts *TransactionSyncer) BroadcastTransaction(tx *blockchain.Transaction) error {
	if !ts.isRunning {
		return fmt.Errorf("交易同步器未运行")
	}

	// 验证交易
	if !ts.validateTransaction(tx) {
		return fmt.Errorf("交易验证失败")
	}

	// 添加到内存池
	if err := ts.addToMempool(tx); err != nil {
		return fmt.Errorf("添加到内存池失败: %v", err)
	}

	// 广播给所有节点
	ts.broadcastTransaction(tx, "")

	return nil
}

// RequestMempool 请求节点的内存池
func (ts *TransactionSyncer) RequestMempool(peerAddr string) error {
	if !ts.isRunning {
		return fmt.Errorf("交易同步器未运行")
	}

	msg := message.NewMessage("mempool", []byte{}, peerAddr)
	msg.Priority = message.PriorityNormal

	return ts.msgHandler.Submit(msg)
}
