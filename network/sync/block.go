package sync

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"mini-coin-go/blockchain"
	"mini-coin-go/network/connection"
	"mini-coin-go/network/message"
)

// BlockSyncer 区块同步器
type BlockSyncer struct {
	blockchain    *blockchain.Blockchain
	connManager   *connection.Manager
	msgHandler    *message.Handler
	maxWorkers    int
	isRunning     bool
	stopCh        chan bool
	mutex         sync.RWMutex
	downloadQueue chan *BlockDownloadTask
	stats         *SyncStats
}

// BlockDownloadTask 区块下载任务
type BlockDownloadTask struct {
	Hash      []byte
	Height    int
	PeerAddr  string
	Retries   int
	CreatedAt time.Time
}

// SyncStats 同步统计信息
type SyncStats struct {
	TotalBlocks       int64         // 总区块数
	DownloadedBlocks  int64         // 已下载区块数
	FailedBlocks      int64         // 失败区块数
	StartTime         time.Time     // 开始时间
	LastBlockTime     time.Time     // 最后区块时间
	AverageBlockTime  time.Duration // 平均区块时间
	DownloadSpeed     float64       // 下载速度(块/秒)
	mutex             sync.RWMutex  // 读写锁
}

// NewBlockSyncer 创建区块同步器
func NewBlockSyncer(bc *blockchain.Blockchain, connManager *connection.Manager, 
	msgHandler *message.Handler, maxWorkers int) *BlockSyncer {
	
	if maxWorkers <= 0 {
		maxWorkers = 10
	}
	
	syncer := &BlockSyncer{
		blockchain:    bc,
		connManager:   connManager,
		msgHandler:    msgHandler,
		maxWorkers:    maxWorkers,
		stopCh:        make(chan bool),
		downloadQueue: make(chan *BlockDownloadTask, 1000),
		stats:         &SyncStats{StartTime: time.Now()},
	}
	
	// 注册消息处理器
	syncer.registerHandlers()
	
	return syncer
}

// Start 启动区块同步器
func (bs *BlockSyncer) Start() error {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()
	
	if bs.isRunning {
		return fmt.Errorf("区块同步器已经在运行")
	}
	
	bs.isRunning = true
	log.Printf("启动区块同步器，最大工作协程数: %d", bs.maxWorkers)
	
	// 启动下载工作协程
	for i := 0; i < bs.maxWorkers; i++ {
		go bs.downloadWorker(i)
	}
	
	return nil
}

// Stop 停止区块同步器
func (bs *BlockSyncer) Stop() error {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()
	
	if !bs.isRunning {
		return nil
	}
	
	bs.isRunning = false
	close(bs.stopCh)
	
	log.Println("区块同步器已停止")
	return nil
}

// registerHandlers 注册消息处理器
func (bs *BlockSyncer) registerHandlers() {
	bs.msgHandler.RegisterHandler("block", bs.handleBlockMessage)
	bs.msgHandler.RegisterHandler("inv", bs.handleInvMessage)
	bs.msgHandler.RegisterHandler("getdata", bs.handleGetDataMessage)
}

// SyncFromPeer 从指定节点同步区块
func (bs *BlockSyncer) SyncFromPeer(peerAddr string) error {
	if !bs.isRunning {
		return fmt.Errorf("区块同步器未运行")
	}
	
	log.Printf("开始从节点同步区块: %s", peerAddr)
	
	// 获取本地最佳高度
	localHeight := bs.blockchain.GetBestHeight()
	
	// 请求节点的区块列表
	return bs.requestBlocksFromPeer(peerAddr, localHeight)
}

// SyncFromMultiplePeers 从多个节点并行同步
func (bs *BlockSyncer) SyncFromMultiplePeers(peerAddrs []string) error {
	if !bs.isRunning {
		return fmt.Errorf("区块同步器未运行")
	}
	
	log.Printf("开始从多个节点同步区块: %v", peerAddrs)
	
	var wg sync.WaitGroup
	errors := make(chan error, len(peerAddrs))
	
	for _, peerAddr := range peerAddrs {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			if err := bs.SyncFromPeer(addr); err != nil {
				errors <- fmt.Errorf("从 %s 同步失败: %v", addr, err)
			}
		}(peerAddr)
	}
	
	// 等待所有同步完成
	go func() {
		wg.Wait()
		close(errors)
	}()
	
	// 收集错误
	var syncErrors []error
	for err := range errors {
		syncErrors = append(syncErrors, err)
	}
	
	if len(syncErrors) > 0 {
		log.Printf("部分节点同步失败: %v", syncErrors)
	}
	
	return nil
}

// requestBlocksFromPeer 从节点请求区块
func (bs *BlockSyncer) requestBlocksFromPeer(peerAddr string, fromHeight int) error {
	// 创建获取区块请求消息
	payload := fmt.Sprintf(`{"from_height": %d}`, fromHeight)
	msg := message.NewMessage("getblocks", []byte(payload), peerAddr)
	msg.Priority = message.PriorityHigh
	
	return bs.msgHandler.Submit(msg)
}

// handleInvMessage 处理库存消息
func (bs *BlockSyncer) handleInvMessage(msg *message.Message) error {
	log.Printf("收到库存消息从 %s", msg.TargetAddr)
	
	// 解析库存消息，获取区块哈希列表
	blockHashes := bs.parseInvMessage(msg.Payload)
	
	// 为每个区块创建下载任务
	for i, hash := range blockHashes {
		task := &BlockDownloadTask{
			Hash:      hash,
			Height:    i + 1, // 这里应该从消息中获取实际高度
			PeerAddr:  msg.TargetAddr,
			Retries:   0,
			CreatedAt: time.Now(),
		}
		
		select {
		case bs.downloadQueue <- task:
			// 任务已加入队列
		default:
			log.Printf("下载队列已满，跳过区块: %x", hash)
		}
	}
	
	return nil
}

// handleBlockMessage 处理区块消息
func (bs *BlockSyncer) handleBlockMessage(msg *message.Message) error {
	start := time.Now()
	
	log.Printf("收到区块消息从 %s", msg.TargetAddr)
	
	// 反序列化区块
	block := blockchain.DeserializeBlock(msg.Payload)
	
	// 验证区块
	if !bs.validateBlock(block) {
		return fmt.Errorf("区块验证失败: %x", block.Hash)
	}
	
	// 添加到区块链
	if err := bs.blockchain.AddBlock(block); err != nil {
		return fmt.Errorf("添加区块失败: %v", err)
	}
	
	// 更新统计信息
	bs.updateStats(time.Since(start))
	
	log.Printf("成功添加区块: %x", block.Hash)
	return nil
}

// handleGetDataMessage 处理获取数据消息
func (bs *BlockSyncer) handleGetDataMessage(msg *message.Message) error {
	log.Printf("收到获取数据消息从 %s", msg.TargetAddr)
	
	// 解析请求的数据类型和ID
	dataType, dataID := bs.parseGetDataMessage(msg.Payload)
	
	if dataType == "block" {
		// 获取区块并发送
		return bs.sendBlockToPeer(dataID, msg.TargetAddr)
	}
	
	return fmt.Errorf("不支持的数据类型: %s", dataType)
}

// downloadWorker 下载工作协程
func (bs *BlockSyncer) downloadWorker(workerID int) {
	log.Printf("下载工作协程 %d 已启动", workerID)
	defer log.Printf("下载工作协程 %d 已停止", workerID)
	
	for {
		select {
		case <-bs.stopCh:
			return
		case task := <-bs.downloadQueue:
			bs.processDownloadTask(task, workerID)
		}
	}
}

// processDownloadTask 处理下载任务
func (bs *BlockSyncer) processDownloadTask(task *BlockDownloadTask, workerID int) {
	log.Printf("工作协程 %d 下载区块: %x", workerID, task.Hash)
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// 创建获取数据消息
	payload := fmt.Sprintf(`{"type": "block", "id": "%x"}`, task.Hash)
	msg := message.NewMessage("getdata", []byte(payload), task.PeerAddr)
	
	// 提交消息进行处理
	if err := bs.msgHandler.Submit(msg); err != nil {
		log.Printf("工作协程 %d 下载区块失败 %x: %v", workerID, task.Hash, err)
		
		// 重试逻辑
		if task.Retries < 3 {
			task.Retries++
			select {
			case bs.downloadQueue <- task:
				// 重新加入队列
			default:
				log.Printf("下载队列已满，放弃重试: %x", task.Hash)
			}
		}
		return
	}
	
	// 等待处理完成或超时
	select {
	case <-ctx.Done():
		log.Printf("工作协程 %d 下载区块超时: %x", workerID, task.Hash)
	case <-time.After(100 * time.Millisecond):
		// 继续处理下一个任务
	}
}

// validateBlock 验证区块
func (bs *BlockSyncer) validateBlock(block *blockchain.Block) bool {
	// 这里应该实现完整的区块验证逻辑
	// 包括：工作量证明验证、交易验证、前一个区块哈希验证等
	
	// 简单验证：检查区块是否为空
	return block != nil && len(block.Hash) > 0
}

// parseInvMessage 解析库存消息
func (bs *BlockSyncer) parseInvMessage(payload []byte) [][]byte {
	// 这里应该解析实际的库存消息格式
	// 暂时返回模拟数据
	var hashes [][]byte
	// 模拟解析逻辑...
	return hashes
}

// parseGetDataMessage 解析获取数据消息
func (bs *BlockSyncer) parseGetDataMessage(payload []byte) (string, []byte) {
	// 这里应该解析实际的获取数据消息格式
	// 暂时返回模拟数据
	return "block", []byte("dummy_hash")
}

// sendBlockToPeer 向节点发送区块
func (bs *BlockSyncer) sendBlockToPeer(blockHash []byte, peerAddr string) error {
	// 从区块链获取区块
	block, err := bs.blockchain.GetBlock(blockHash)
	if err != nil {
		return fmt.Errorf("获取区块失败: %v", err)
	}
	
	// 序列化区块
	blockData := block.Serialize()
	
	// 创建区块消息
	msg := message.NewMessage("block", blockData, peerAddr)
	msg.Priority = message.PriorityHigh
	
	return bs.msgHandler.Submit(msg)
}

// updateStats 更新统计信息
func (bs *BlockSyncer) updateStats(blockTime time.Duration) {
	bs.stats.mutex.Lock()
	defer bs.stats.mutex.Unlock()
	
	bs.stats.DownloadedBlocks++
	bs.stats.LastBlockTime = time.Now()
	
	// 更新平均区块时间
	if bs.stats.AverageBlockTime == 0 {
		bs.stats.AverageBlockTime = blockTime
	} else {
		bs.stats.AverageBlockTime = (bs.stats.AverageBlockTime + blockTime) / 2
	}
	
	// 计算下载速度
	elapsed := time.Since(bs.stats.StartTime).Seconds()
	if elapsed > 0 {
		bs.stats.DownloadSpeed = float64(bs.stats.DownloadedBlocks) / elapsed
	}
}

// GetStats 获取同步统计信息
func (bs *BlockSyncer) GetStats() *SyncStats {
	bs.stats.mutex.RLock()
	defer bs.stats.mutex.RUnlock()
	
	return &SyncStats{
		TotalBlocks:      bs.stats.TotalBlocks,
		DownloadedBlocks: bs.stats.DownloadedBlocks,
		FailedBlocks:     bs.stats.FailedBlocks,
		StartTime:        bs.stats.StartTime,
		LastBlockTime:    bs.stats.LastBlockTime,
		AverageBlockTime: bs.stats.AverageBlockTime,
		DownloadSpeed:    bs.stats.DownloadSpeed,
	}
}

// GetSyncProgress 获取同步进度
func (bs *BlockSyncer) GetSyncProgress() map[string]interface{} {
	stats := bs.GetStats()
	
	progress := float64(0)
	if stats.TotalBlocks > 0 {
		progress = float64(stats.DownloadedBlocks) / float64(stats.TotalBlocks) * 100
	}
	
	return map[string]interface{}{
		"is_running":         bs.isRunning,
		"total_blocks":       stats.TotalBlocks,
		"downloaded_blocks":  stats.DownloadedBlocks,
		"failed_blocks":      stats.FailedBlocks,
		"progress_percent":   progress,
		"download_speed":     stats.DownloadSpeed,
		"average_block_time": stats.AverageBlockTime.String(),
		"queue_size":         len(bs.downloadQueue),
	}
}

// IsRunning 检查是否正在运行
func (bs *BlockSyncer) IsRunning() bool {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()
	return bs.isRunning
}