package message

import (
	"container/heap"
	"sync"
	"time"
)

// Priority 消息优先级枚举
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// String 返回优先级字符串表示
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Message 消息结构
type Message struct {
	ID          string      `json:"id"`           // 消息ID
	Type        string      `json:"type"`         // 消息类型
	Priority    Priority    `json:"priority"`     // 优先级
	Payload     []byte      `json:"payload"`      // 消息内容
	TargetAddr  string      `json:"target_addr"`  // 目标地址
	CreatedAt   time.Time   `json:"created_at"`   // 创建时间
	Retries     int         `json:"retries"`      // 重试次数
	MaxRetries  int         `json:"max_retries"`  // 最大重试次数
	Timeout     time.Duration `json:"timeout"`    // 超时时间
}

// NewMessage 创建新消息
func NewMessage(msgType string, payload []byte, targetAddr string) *Message {
	return &Message{
		ID:         generateMessageID(),
		Type:       msgType,
		Priority:   PriorityNormal,
		Payload:    payload,
		TargetAddr: targetAddr,
		CreatedAt:  time.Now(),
		Retries:    0,
		MaxRetries: 3,
		Timeout:    30 * time.Second,
	}
}

// generateMessageID 生成消息ID
func generateMessageID() string {
	return time.Now().Format("20060102150405.000000")
}

// IsExpired 检查消息是否过期
func (m *Message) IsExpired() bool {
	return time.Since(m.CreatedAt) > m.Timeout
}

// CanRetry 检查是否可以重试
func (m *Message) CanRetry() bool {
	return m.Retries < m.MaxRetries && !m.IsExpired()
}

// IncrementRetries 增加重试次数
func (m *Message) IncrementRetries() {
	m.Retries++
}

// PriorityQueue 优先级队列
type PriorityQueue []*Message

// Len 返回队列长度
func (pq PriorityQueue) Len() int {
	return len(pq)
}

// Less 比较优先级
func (pq PriorityQueue) Less(i, j int) bool {
	// 优先级高的排在前面
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority > pq[j].Priority
	}
	// 优先级相同时，创建时间早的排在前面
	return pq[i].CreatedAt.Before(pq[j].CreatedAt)
}

// Swap 交换元素
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

// Push 添加元素
func (pq *PriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*Message))
}

// Pop 弹出元素
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

// Queue 消息队列
type Queue struct {
	messages    PriorityQueue          // 优先级队列
	pending     map[string]*Message    // 待处理消息映射
	processing  map[string]*Message    // 处理中消息映射
	failed      map[string]*Message    // 失败消息映射
	mutex       sync.RWMutex           // 读写锁
	maxSize     int                    // 最大队列大小
	stats       *QueueStats            // 统计信息
}

// QueueStats 队列统计信息
type QueueStats struct {
	TotalMessages      int64         // 总消息数
	ProcessedMessages  int64         // 已处理消息数
	FailedMessages     int64         // 失败消息数
	PendingMessages    int           // 待处理消息数
	ProcessingMessages int           // 处理中消息数
	AverageProcessTime time.Duration // 平均处理时间
	mutex             sync.RWMutex   // 读写锁
}

// NewQueue 创建消息队列
func NewQueue(maxSize int) *Queue {
	if maxSize <= 0 {
		maxSize = 1000
	}
	
	queue := &Queue{
		messages:   make(PriorityQueue, 0),
		pending:    make(map[string]*Message),
		processing: make(map[string]*Message),
		failed:     make(map[string]*Message),
		maxSize:    maxSize,
		stats:      &QueueStats{},
	}
	
	heap.Init(&queue.messages)
	return queue
}

// Enqueue 添加消息到队列
func (q *Queue) Enqueue(message *Message) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	
	// 检查队列是否已满
	if len(q.messages) >= q.maxSize {
		return NewQueueError("队列已满", ErrQueueFull)
	}
	
	// 检查消息是否已存在
	if _, exists := q.pending[message.ID]; exists {
		return NewQueueError("消息已存在", ErrMessageExists)
	}
	
	heap.Push(&q.messages, message)
	q.pending[message.ID] = message
	
	q.stats.mutex.Lock()
	q.stats.TotalMessages++
	q.stats.PendingMessages++
	q.stats.mutex.Unlock()
	
	return nil
}

// Dequeue 从队列取出消息
func (q *Queue) Dequeue() *Message {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	
	if q.messages.Len() == 0 {
		return nil
	}
	
	message := heap.Pop(&q.messages).(*Message)
	delete(q.pending, message.ID)
	q.processing[message.ID] = message
	
	q.stats.mutex.Lock()
	q.stats.PendingMessages--
	q.stats.ProcessingMessages++
	q.stats.mutex.Unlock()
	
	return message
}

// MarkProcessed 标记消息已处理
func (q *Queue) MarkProcessed(messageID string, processTime time.Duration) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	
	if message, exists := q.processing[messageID]; exists {
		delete(q.processing, messageID)
		
		q.stats.mutex.Lock()
		q.stats.ProcessedMessages++
		q.stats.ProcessingMessages--
		q.updateAverageProcessTime(processTime)
		q.stats.mutex.Unlock()
		
		_ = message // 使用变量避免编译警告
	}
}

// MarkFailed 标记消息处理失败
func (q *Queue) MarkFailed(messageID string, reason string) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	
	if message, exists := q.processing[messageID]; exists {
		delete(q.processing, messageID)
		message.IncrementRetries()
		
		if message.CanRetry() {
			// 重新加入队列
			heap.Push(&q.messages, message)
			q.pending[message.ID] = message
			q.stats.mutex.Lock()
			q.stats.PendingMessages++
			q.stats.ProcessingMessages--
			q.stats.mutex.Unlock()
		} else {
			// 标记为失败
			q.failed[message.ID] = message
			q.stats.mutex.Lock()
			q.stats.FailedMessages++
			q.stats.ProcessingMessages--
			q.stats.mutex.Unlock()
		}
	}
}

// GetPendingCount 获取待处理消息数量
func (q *Queue) GetPendingCount() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return len(q.messages)
}

// GetProcessingCount 获取处理中消息数量
func (q *Queue) GetProcessingCount() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return len(q.processing)
}

// GetFailedCount 获取失败消息数量
func (q *Queue) GetFailedCount() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return len(q.failed)
}

// GetStats 获取队列统计信息
func (q *Queue) GetStats() *QueueStats {
	q.stats.mutex.RLock()
	defer q.stats.mutex.RUnlock()
	
	return &QueueStats{
		TotalMessages:      q.stats.TotalMessages,
		ProcessedMessages:  q.stats.ProcessedMessages,
		FailedMessages:     q.stats.FailedMessages,
		PendingMessages:    q.GetPendingCount(),
		ProcessingMessages: q.GetProcessingCount(),
		AverageProcessTime: q.stats.AverageProcessTime,
	}
}

// updateAverageProcessTime 更新平均处理时间
func (q *Queue) updateAverageProcessTime(duration time.Duration) {
	if q.stats.AverageProcessTime == 0 {
		q.stats.AverageProcessTime = duration
	} else {
		// 简单的移动平均
		q.stats.AverageProcessTime = (q.stats.AverageProcessTime + duration) / 2
	}
}

// Clear 清空队列
func (q *Queue) Clear() {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	
	q.messages = make(PriorityQueue, 0)
	heap.Init(&q.messages)
	q.pending = make(map[string]*Message)
	q.processing = make(map[string]*Message)
	q.failed = make(map[string]*Message)
	
	q.stats.mutex.Lock()
	q.stats.PendingMessages = 0
	q.stats.ProcessingMessages = 0
	q.stats.mutex.Unlock()
}

// GetFailedMessages 获取所有失败的消息
func (q *Queue) GetFailedMessages() []*Message {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	
	var failed []*Message
	for _, message := range q.failed {
		failed = append(failed, message)
	}
	return failed
}

// RetryFailedMessage 重试失败的消息
func (q *Queue) RetryFailedMessage(messageID string) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	
	message, exists := q.failed[messageID]
	if !exists {
		return NewQueueError("消息不存在", ErrMessageNotFound)
	}
	
	// 重置重试次数
	message.Retries = 0
	
	// 重新加入队列
	delete(q.failed, messageID)
	heap.Push(&q.messages, message)
	q.pending[message.ID] = message
	
	q.stats.mutex.Lock()
	q.stats.FailedMessages--
	q.stats.PendingMessages++
	q.stats.mutex.Unlock()
	
	return nil
}

// Cleanup 清理过期消息
func (q *Queue) Cleanup() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	
	cleaned := 0
	
	// 清理过期的失败消息
	for id, message := range q.failed {
		if message.IsExpired() {
			delete(q.failed, id)
			cleaned++
		}
	}
	
	q.stats.mutex.Lock()
	q.stats.FailedMessages -= int64(cleaned)
	q.stats.mutex.Unlock()
	
	return cleaned
}