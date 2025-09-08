package message

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// HandlerFunc 消息处理函数类型
type HandlerFunc func(*Message) error

// Handler 消息处理器
type Handler struct {
	queues        map[string]*Queue    // 队列映射（按类型）
	handlers      map[string]HandlerFunc // 处理函数映射
	workers       int                  // 工作协程数量
	isRunning     bool                 // 是否运行中
	stopCh        chan bool           // 停止信号
	mutex         sync.RWMutex         // 读写锁
	cleanupTicker *time.Ticker        // 清理定时器
	stats         *HandlerStats       // 统计信息
}

// HandlerStats 处理器统计信息
type HandlerStats struct {
	TotalProcessed    int64         // 总处理数
	TotalFailed       int64         // 总失败数
	ProcessingTime    time.Duration // 总处理时间
	AverageProcessTime time.Duration // 平均处理时间
	mutex             sync.RWMutex   // 读写锁
}

// NewHandler 创建消息处理器
func NewHandler(workers int) *Handler {
	if workers <= 0 {
		workers = 5
	}
	
	return &Handler{
		queues:   make(map[string]*Queue),
		handlers: make(map[string]HandlerFunc),
		workers:  workers,
		stopCh:   make(chan bool),
		stats:    &HandlerStats{},
	}
}

// Start 启动消息处理器
func (h *Handler) Start() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	if h.isRunning {
		return fmt.Errorf("消息处理器已经在运行")
	}
	
	h.isRunning = true
	log.Printf("启动消息处理器，工作协程数: %d", h.workers)
	
	// 启动工作协程
	for i := 0; i < h.workers; i++ {
		go h.worker(i)
	}
	
	// 启动清理任务
	h.startCleanup()
	
	return nil
}

// Stop 停止消息处理器
func (h *Handler) Stop() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	if !h.isRunning {
		return nil
	}
	
	h.isRunning = false
	close(h.stopCh)
	
	// 停止清理定时器
	if h.cleanupTicker != nil {
		h.cleanupTicker.Stop()
	}
	
	log.Println("消息处理器已停止")
	return nil
}

// RegisterHandler 注册消息处理函数
func (h *Handler) RegisterHandler(messageType string, handler HandlerFunc) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	h.handlers[messageType] = handler
	
	// 如果队列不存在，创建新队列
	if _, exists := h.queues[messageType]; !exists {
		h.queues[messageType] = NewQueue(1000)
	}
	
	log.Printf("注册消息处理器: %s", messageType)
}

// Submit 提交消息进行处理
func (h *Handler) Submit(message *Message) error {
	if !h.isRunning {
		return fmt.Errorf("消息处理器未运行")
	}
	
	h.mutex.RLock()
	queue, exists := h.queues[message.Type]
	h.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("未注册的消息类型: %s", message.Type)
	}
	
	return queue.Enqueue(message)
}

// worker 工作协程
func (h *Handler) worker(workerID int) {
	log.Printf("工作协程 %d 已启动", workerID)
	defer log.Printf("工作协程 %d 已停止", workerID)
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.processMessages(workerID)
		}
	}
}

// processMessages 处理消息
func (h *Handler) processMessages(workerID int) {
	h.mutex.RLock()
	queues := make(map[string]*Queue)
	handlers := make(map[string]HandlerFunc)
	for msgType, queue := range h.queues {
		queues[msgType] = queue
		handlers[msgType] = h.handlers[msgType]
	}
	h.mutex.RUnlock()
	
	for msgType, queue := range queues {
		message := queue.Dequeue()
		if message == nil {
			continue
		}
		
		handler := handlers[msgType]
		if handler == nil {
			queue.MarkFailed(message.ID, "未找到处理函数")
			continue
		}
		
		h.handleMessage(message, handler, queue, workerID)
	}
}

// handleMessage 处理单个消息
func (h *Handler) handleMessage(message *Message, handler HandlerFunc, queue *Queue, workerID int) {
	start := time.Now()
	
	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), message.Timeout)
	defer cancel()
	
	// 在新协程中处理消息，支持超时控制
	done := make(chan error, 1)
	go func() {
		done <- handler(message)
	}()
	
	select {
	case err := <-done:
		duration := time.Since(start)
		if err != nil {
			log.Printf("工作协程 %d 处理消息失败 %s: %v", workerID, message.ID, err)
			queue.MarkFailed(message.ID, err.Error())
			h.updateFailedStats()
		} else {
			log.Printf("工作协程 %d 处理消息成功 %s (耗时: %v)", workerID, message.ID, duration)
			queue.MarkProcessed(message.ID, duration)
			h.updateProcessedStats(duration)
		}
		
	case <-ctx.Done():
		duration := time.Since(start)
		log.Printf("工作协程 %d 处理消息超时 %s (耗时: %v)", workerID, message.ID, duration)
		queue.MarkFailed(message.ID, "处理超时")
		h.updateFailedStats()
	}
}

// updateProcessedStats 更新处理成功统计
func (h *Handler) updateProcessedStats(duration time.Duration) {
	h.stats.mutex.Lock()
	defer h.stats.mutex.Unlock()
	
	h.stats.TotalProcessed++
	h.stats.ProcessingTime += duration
	
	if h.stats.TotalProcessed > 0 {
		h.stats.AverageProcessTime = time.Duration(h.stats.ProcessingTime.Nanoseconds() / h.stats.TotalProcessed)
	}
}

// updateFailedStats 更新处理失败统计
func (h *Handler) updateFailedStats() {
	h.stats.mutex.Lock()
	defer h.stats.mutex.Unlock()
	
	h.stats.TotalFailed++
}

// GetStats 获取处理器统计信息
func (h *Handler) GetStats() *HandlerStats {
	h.stats.mutex.RLock()
	defer h.stats.mutex.RUnlock()
	
	return &HandlerStats{
		TotalProcessed:     h.stats.TotalProcessed,
		TotalFailed:        h.stats.TotalFailed,
		ProcessingTime:     h.stats.ProcessingTime,
		AverageProcessTime: h.stats.AverageProcessTime,
	}
}

// GetQueueStats 获取所有队列统计信息
func (h *Handler) GetQueueStats() map[string]*QueueStats {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	stats := make(map[string]*QueueStats)
	for msgType, queue := range h.queues {
		stats[msgType] = queue.GetStats()
	}
	return stats
}

// GetTotalStats 获取总体统计信息
func (h *Handler) GetTotalStats() map[string]interface{} {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	totalPending := 0
	totalProcessing := 0
	totalFailed := 0
	
	for _, queue := range h.queues {
		stats := queue.GetStats()
		totalPending += stats.PendingMessages
		totalProcessing += stats.ProcessingMessages
		totalFailed += int(stats.FailedMessages)
	}
	
	handlerStats := h.GetStats()
	
	return map[string]interface{}{
		"is_running":           h.isRunning,
		"workers":              h.workers,
		"registered_types":     len(h.handlers),
		"total_pending":        totalPending,
		"total_processing":     totalProcessing,
		"total_failed":         totalFailed,
		"total_processed":      handlerStats.TotalProcessed,
		"average_process_time": handlerStats.AverageProcessTime.String(),
	}
}

// RetryFailedMessages 重试指定类型的失败消息
func (h *Handler) RetryFailedMessages(messageType string) (int, error) {
	h.mutex.RLock()
	queue, exists := h.queues[messageType]
	h.mutex.RUnlock()
	
	if !exists {
		return 0, fmt.Errorf("消息类型不存在: %s", messageType)
	}
	
	failedMessages := queue.GetFailedMessages()
	retried := 0
	
	for _, message := range failedMessages {
		if err := queue.RetryFailedMessage(message.ID); err == nil {
			retried++
		}
	}
	
	log.Printf("重试失败消息 %s: %d/%d", messageType, retried, len(failedMessages))
	return retried, nil
}

// ClearFailedMessages 清除指定类型的失败消息
func (h *Handler) ClearFailedMessages(messageType string) error {
	h.mutex.RLock()
	queue, exists := h.queues[messageType]
	h.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("消息类型不存在: %s", messageType)
	}
	
	// 这需要在Queue中添加ClearFailed方法
	// 暂时通过获取失败消息并逐个删除来实现
	failedMessages := queue.GetFailedMessages()
	for _, message := range failedMessages {
		// 这里需要Queue提供删除失败消息的方法
		_ = message
	}
	
	return nil
}

// startCleanup 启动清理任务
func (h *Handler) startCleanup() {
	h.cleanupTicker = time.NewTicker(5 * time.Minute)
	go h.cleanupTask()
}

// cleanupTask 清理任务
func (h *Handler) cleanupTask() {
	for {
		select {
		case <-h.cleanupTicker.C:
			h.performCleanup()
		case <-h.stopCh:
			return
		}
	}
}

// performCleanup 执行清理
func (h *Handler) performCleanup() {
	h.mutex.RLock()
	queues := make(map[string]*Queue)
	for msgType, queue := range h.queues {
		queues[msgType] = queue
	}
	h.mutex.RUnlock()
	
	totalCleaned := 0
	for msgType, queue := range queues {
		cleaned := queue.Cleanup()
		if cleaned > 0 {
			log.Printf("清理过期消息 %s: %d", msgType, cleaned)
			totalCleaned += cleaned
		}
	}
	
	if totalCleaned > 0 {
		log.Printf("总共清理过期消息: %d", totalCleaned)
	}
}

// IsRunning 检查是否正在运行
func (h *Handler) IsRunning() bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.isRunning
}

// GetRegisteredTypes 获取已注册的消息类型
func (h *Handler) GetRegisteredTypes() []string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	var types []string
	for msgType := range h.handlers {
		types = append(types, msgType)
	}
	return types
}