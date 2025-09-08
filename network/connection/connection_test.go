package connection

import (
	"context"
	"testing"
	"time"
)

// TestConnectionPool 测试连接池
func TestConnectionPool(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxConnections = 2
	config.MaxIdleTime = 1 * time.Second

	pool := NewPool("localhost:9999", config) // 使用不存在的地址

	t.Run("PoolLifecycle", func(t *testing.T) {
		// 启动连接池
		err := pool.Start()
		if err != nil {
			t.Errorf("Failed to start pool: %v", err)
		}

		// 检查统计信息
		stats := pool.GetStats()
		if stats.TotalConnections != 0 {
			t.Errorf("Expected 0 total connections, got %d", stats.TotalConnections)
		}

		// 停止连接池
		err = pool.Stop()
		if err != nil {
			t.Errorf("Failed to stop pool: %v", err)
		}
	})

	t.Run("ConnectionOperations", func(t *testing.T) {
		err := pool.Start()
		if err != nil {
			t.Errorf("Failed to start pool: %v", err)
		}
		defer pool.Stop()

		// 尝试获取连接（应该失败，因为地址不存在）
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		_, err = pool.GetConnection(ctx)
		if err == nil {
			t.Error("Expected error when connecting to invalid address")
		}
	})
}

// TestPoolConfig 测试连接池配置
func TestPoolConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultPoolConfig()

		if config.MaxConnections <= 0 {
			t.Error("MaxConnections should be positive")
		}

		if config.MaxIdleTime <= 0 {
			t.Error("MaxIdleTime should be positive")
		}

		if config.ConnectTimeout <= 0 {
			t.Error("ConnectTimeout should be positive")
		}
	})

	t.Run("CustomConfig", func(t *testing.T) {
		config := &PoolConfig{
			MaxConnections: 5,
			MaxIdleTime:    30 * time.Second,
			ConnectTimeout: 10 * time.Second,
		}

		if config.MaxConnections != 5 {
			t.Errorf("Expected MaxConnections 5, got %d", config.MaxConnections)
		}
	})
}
