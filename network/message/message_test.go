package message

import (
	"fmt"
	"testing"
	"time"
)

// TestMessageQueue 测试消息队列
func TestMessageQueue(t *testing.T) {
	queue := NewQueue(10)

	t.Run("BasicQueueOperations", func(t *testing.T) {
		// 测试入队
		msg := &Message{
			ID:       "test-1",
			Type:     "test",
			Payload:  []byte("test data"),
			Priority: PriorityNormal,
			Timeout:  5 * time.Second,
		}

		err := queue.Enqueue(msg)
		if err != nil {
			t.Errorf("Failed to enqueue message: %v", err)
		}

		// 测试出队
		dequeued := queue.Dequeue()
		if dequeued == nil {
			t.Error("Expected message, got nil")
		}

		if dequeued.ID != msg.ID {
			t.Errorf("Expected message ID %s, got %s", msg.ID, dequeued.ID)
		}
	})

	t.Run("PriorityOrdering", func(t *testing.T) {
		// 添加不同优先级的消息
		highMsg := &Message{
			ID:       "high-priority",
			Type:     "test",
			Priority: PriorityHigh,
			Timeout:  5 * time.Second,
		}

		normalMsg := &Message{
			ID:       "normal-priority",
			Type:     "test",
			Priority: PriorityNormal,
			Timeout:  5 * time.Second,
		}

		lowMsg := &Message{
			ID:       "low-priority",
			Type:     "test",
			Priority: PriorityLow,
			Timeout:  5 * time.Second,
		}

		// 以低优先级顺序入队
		queue.Enqueue(lowMsg)
		queue.Enqueue(normalMsg)
		queue.Enqueue(highMsg)

		// 应该按高优先级顺序出队
		first := queue.Dequeue()
		if first == nil || first.ID != "high-priority" {
			t.Error("High priority message should be dequeued first")
		}

		second := queue.Dequeue()
		if second == nil || second.ID != "normal-priority" {
			t.Error("Normal priority message should be dequeued second")
		}

		third := queue.Dequeue()
		if third == nil || third.ID != "low-priority" {
			t.Error("Low priority message should be dequeued third")
		}
	})

	t.Run("QueueStats", func(t *testing.T) {
		// 清空队列
		for queue.Dequeue() != nil {
		}

		stats := queue.GetStats()
		if stats.PendingMessages != 0 {
			t.Errorf("Expected 0 pending messages, got %d", stats.PendingMessages)
		}

		// 添加消息
		msg := &Message{
			ID:       "stats-test",
			Type:     "test",
			Priority: PriorityNormal,
			Timeout:  5 * time.Second,
		}
		queue.Enqueue(msg)

		stats = queue.GetStats()
		if stats.PendingMessages != 1 {
			t.Errorf("Expected 1 pending message, got %d", stats.PendingMessages)
		}
	})

	t.Run("MessageProcessing", func(t *testing.T) {
		msg := &Message{
			ID:       "process-test",
			Type:     "test",
			Priority: PriorityNormal,
			Timeout:  5 * time.Second,
		}

		queue.Enqueue(msg)
		retrieved := queue.Dequeue()

		// 标记为已处理
		queue.MarkProcessed(retrieved.ID, 100*time.Millisecond)

		stats := queue.GetStats()
		if stats.ProcessedMessages != 1 {
			t.Errorf("Expected 1 processed message, got %d", stats.ProcessedMessages)
		}
	})

	t.Run("MessageFailure", func(t *testing.T) {
		msg := &Message{
			ID:       "fail-test",
			Type:     "test",
			Priority: PriorityNormal,
			Timeout:  5 * time.Second,
		}

		queue.Enqueue(msg)
		retrieved := queue.Dequeue()

		// 标记为失败
		queue.MarkFailed(retrieved.ID, "test error")

		stats := queue.GetStats()
		if stats.FailedMessages != 1 {
			t.Errorf("Expected 1 failed message, got %d", stats.FailedMessages)
		}

		// 获取失败的消息
		failedMessages := queue.GetFailedMessages()
		if len(failedMessages) != 1 {
			t.Errorf("Expected 1 failed message in list, got %d", len(failedMessages))
		}
	})

	t.Run("QueueCapacity", func(t *testing.T) {
		smallQueue := NewQueue(2)

		// 填满队列
		for i := 0; i < 2; i++ {
			msg := &Message{
				ID:       fmt.Sprintf("capacity-test-%d", i),
				Type:     "test",
				Priority: PriorityNormal,
				Timeout:  5 * time.Second,
			}
			smallQueue.Enqueue(msg)
		}

		// 尝试添加超出容量的消息
		extraMsg := &Message{
			ID:       "extra-msg",
			Type:     "test",
			Priority: PriorityNormal,
			Timeout:  5 * time.Second,
		}

		err := smallQueue.Enqueue(extraMsg)
		if err == nil {
			t.Error("Expected error when exceeding queue capacity")
		}
	})
}

// TestMessageCreation 测试消息创建
func TestMessageCreation(t *testing.T) {
	t.Run("NewMessage", func(t *testing.T) {
		msg := NewMessage("test", []byte("test data"), "target-address")

		if msg.Type != "test" {
			t.Errorf("Expected type 'test', got '%s'", msg.Type)
		}

		if string(msg.Payload) != "test data" {
			t.Errorf("Expected payload 'test data', got '%s'", string(msg.Payload))
		}

		if msg.TargetAddr != "target-address" {
			t.Errorf("Expected target address 'target-address', got '%s'", msg.TargetAddr)
		}

		if msg.Priority != PriorityNormal {
			t.Errorf("Expected priority %d, got %d", PriorityNormal, msg.Priority)
		}

		if msg.ID == "" {
			t.Error("Message ID should not be empty")
		}
	})

	t.Run("MessageTimeout", func(t *testing.T) {
		msg := NewMessage("test", []byte("data"), "addr")

		// 默认超时应该是合理的值
		if msg.Timeout <= 0 {
			t.Error("Message timeout should be positive")
		}

		if msg.Timeout > 1*time.Minute {
			t.Error("Default timeout seems too long")
		}
	})
}
