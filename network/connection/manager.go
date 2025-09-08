package connection

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Manager 连接管理器
type Manager struct {
	pools       map[string]*Pool    // 地址 -> 连接池映射
	config      *PoolConfig         // 默认配置
	mutex       sync.RWMutex        // 读写锁
	isRunning   bool               // 是否运行中
}

// NewManager 创建连接管理器
func NewManager(config *PoolConfig) *Manager {
	if config == nil {
		config = DefaultPoolConfig()
	}
	
	return &Manager{
		pools:     make(map[string]*Pool),
		config:    config,
		isRunning: false,
	}
}

// Start 启动连接管理器
func (m *Manager) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.isRunning {
		return fmt.Errorf("连接管理器已经在运行")
	}
	
	m.isRunning = true
	log.Println("连接管理器已启动")
	
	return nil
}

// Stop 停止连接管理器
func (m *Manager) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if !m.isRunning {
		return nil
	}
	
	m.isRunning = false
	
	// 停止所有连接池
	for address, pool := range m.pools {
		if err := pool.Stop(); err != nil {
			log.Printf("停止连接池失败 %s: %v", address, err)
		}
	}
	
	m.pools = make(map[string]*Pool)
	log.Println("连接管理器已停止")
	
	return nil
}

// GetConnection 获取到指定地址的连接
func (m *Manager) GetConnection(ctx context.Context, address string) (*Connection, error) {
	if !m.isRunning {
		return nil, fmt.Errorf("连接管理器未运行")
	}
	
	pool, err := m.getOrCreatePool(address)
	if err != nil {
		return nil, fmt.Errorf("获取连接池失败: %v", err)
	}
	
	return pool.GetConnection(ctx)
}

// ReturnConnection 归还连接
func (m *Manager) ReturnConnection(conn *Connection) {
	if conn == nil {
		return
	}
	
	address := conn.RemoteAddr
	
	m.mutex.RLock()
	pool, exists := m.pools[address]
	m.mutex.RUnlock()
	
	if exists {
		pool.ReturnConnection(conn)
	} else {
		// 连接池不存在，直接关闭连接
		conn.Close()
	}
}

// getOrCreatePool 获取或创建连接池
func (m *Manager) getOrCreatePool(address string) (*Pool, error) {
	// 先尝试获取现有连接池
	m.mutex.RLock()
	if pool, exists := m.pools[address]; exists {
		m.mutex.RUnlock()
		return pool, nil
	}
	m.mutex.RUnlock()
	
	// 需要创建新连接池
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// 双重检查
	if pool, exists := m.pools[address]; exists {
		return pool, nil
	}
	
	// 创建新连接池
	pool := NewPool(address, m.config)
	if err := pool.Start(); err != nil {
		return nil, fmt.Errorf("启动连接池失败: %v", err)
	}
	
	m.pools[address] = pool
	log.Printf("创建新连接池: %s", address)
	
	return pool, nil
}

// RemovePool 移除连接池
func (m *Manager) RemovePool(address string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if pool, exists := m.pools[address]; exists {
		if err := pool.Stop(); err != nil {
			return fmt.Errorf("停止连接池失败: %v", err)
		}
		delete(m.pools, address)
		log.Printf("移除连接池: %s", address)
	}
	
	return nil
}

// GetPoolStats 获取指定地址的连接池统计信息
func (m *Manager) GetPoolStats(address string) *PoolStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if pool, exists := m.pools[address]; exists {
		return pool.GetStats()
	}
	
	return nil
}

// GetAllPoolStats 获取所有连接池的统计信息
func (m *Manager) GetAllPoolStats() map[string]*PoolStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	stats := make(map[string]*PoolStats)
	for address, pool := range m.pools {
		stats[address] = pool.GetStats()
	}
	
	return stats
}

// GetManagerStats 获取管理器统计信息
func (m *Manager) GetManagerStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	totalConnections := 0
	totalActiveConnections := 0
	totalIdleConnections := 0
	
	for _, pool := range m.pools {
		stats := pool.GetStats()
		totalConnections += stats.TotalConnections
		totalActiveConnections += stats.ActiveConnections
		totalIdleConnections += stats.IdleConnections
	}
	
	return map[string]interface{}{
		"is_running":             m.isRunning,
		"total_pools":           len(m.pools),
		"total_connections":     totalConnections,
		"total_active_connections": totalActiveConnections,
		"total_idle_connections":   totalIdleConnections,
		"max_connections_per_pool": m.config.MaxConnections,
	}
}

// ExecuteWithConnection 使用连接执行操作
func (m *Manager) ExecuteWithConnection(ctx context.Context, address string, 
	operation func(*Connection) error) error {
	
	conn, err := m.GetConnection(ctx, address)
	if err != nil {
		return fmt.Errorf("获取连接失败: %v", err)
	}
	defer m.ReturnConnection(conn)
	
	return operation(conn)
}

// ExecuteWithTimeout 带超时的连接操作
func (m *Manager) ExecuteWithTimeout(address string, timeout time.Duration,
	operation func(*Connection) error) error {
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	return m.ExecuteWithConnection(ctx, address, operation)
}

// Ping 测试到指定地址的连接
func (m *Manager) Ping(address string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return m.ExecuteWithConnection(ctx, address, func(conn *Connection) error {
		// 简单的连接测试
		return nil
	})
}

// GetActiveAddresses 获取所有活跃的地址
func (m *Manager) GetActiveAddresses() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	var addresses []string
	for address, pool := range m.pools {
		stats := pool.GetStats()
		if stats.TotalConnections > 0 {
			addresses = append(addresses, address)
		}
	}
	
	return addresses
}

// CleanupIdlePools 清理空闲的连接池
func (m *Manager) CleanupIdlePools() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	var toRemove []string
	
	for address, pool := range m.pools {
		stats := pool.GetStats()
		if stats.TotalConnections == 0 && stats.IdleConnections == 0 {
			toRemove = append(toRemove, address)
		}
	}
	
	for _, address := range toRemove {
		if pool, exists := m.pools[address]; exists {
			pool.Stop()
			delete(m.pools, address)
			log.Printf("清理空闲连接池: %s", address)
		}
	}
	
	if len(toRemove) > 0 {
		log.Printf("清理了 %d 个空闲连接池", len(toRemove))
	}
}

// SetPoolConfig 设置连接池配置
func (m *Manager) SetPoolConfig(config *PoolConfig) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.config = config
}

// GetPoolConfig 获取连接池配置
func (m *Manager) GetPoolConfig() *PoolConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.config
}