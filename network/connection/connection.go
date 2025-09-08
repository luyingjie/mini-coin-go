package connection

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Connection 表示一个网络连接
type Connection struct {
	ID          string        // 连接唯一标识
	Conn        net.Conn      // 底层网络连接
	RemoteAddr  string        // 远程地址
	CreatedAt   time.Time     // 创建时间
	LastUsed    time.Time     // 最后使用时间
	IsActive    bool          // 是否活跃
	IsBusy      bool          // 是否忙碌
	UsageCount  int           // 使用次数
	mutex       sync.RWMutex  // 读写锁
}

// NewConnection 创建新连接
func NewConnection(conn net.Conn) *Connection {
	return &Connection{
		ID:         generateConnectionID(),
		Conn:       conn,
		RemoteAddr: conn.RemoteAddr().String(),
		CreatedAt:  time.Now(),
		LastUsed:   time.Now(),
		IsActive:   true,
		IsBusy:     false,
		UsageCount: 0,
	}
}

// generateConnectionID 生成连接ID
func generateConnectionID() string {
	return fmt.Sprintf("conn_%d", time.Now().UnixNano())
}

// Use 使用连接
func (c *Connection) Use() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.LastUsed = time.Now()
	c.UsageCount++
	c.IsBusy = true
}

// Release 释放连接
func (c *Connection) Release() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.IsBusy = false
	c.LastUsed = time.Now()
}

// Close 关闭连接
func (c *Connection) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.IsActive = false
	if c.Conn != nil {
		return c.Conn.Close()
	}
	return nil
}

// IsExpired 检查连接是否过期
func (c *Connection) IsExpired(maxIdleTime time.Duration) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	return time.Since(c.LastUsed) > maxIdleTime && !c.IsBusy
}

// IsHealthy 检查连接是否健康
func (c *Connection) IsHealthy() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	if !c.IsActive || c.Conn == nil {
		return false
	}
	
	// 设置读取超时
	c.Conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
	
	// 尝试读取一个字节
	one := make([]byte, 1)
	_, err := c.Conn.Read(one)
	
	// 重置读取超时
	c.Conn.SetReadDeadline(time.Time{})
	
	// 如果是超时错误，说明连接健康（没有数据可读）
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	
	// 其他错误或EOF说明连接有问题
	return err == nil
}

// GetInfo 获取连接信息
func (c *Connection) GetInfo() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	return map[string]interface{}{
		"id":          c.ID,
		"remote_addr": c.RemoteAddr,
		"created_at":  c.CreatedAt,
		"last_used":   c.LastUsed,
		"is_active":   c.IsActive,
		"is_busy":     c.IsBusy,
		"usage_count": c.UsageCount,
		"age":         time.Since(c.CreatedAt).Seconds(),
		"idle_time":   time.Since(c.LastUsed).Seconds(),
	}
}

// String 返回连接的字符串表示
func (c *Connection) String() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	return fmt.Sprintf("Connection{ID: %s, RemoteAddr: %s, Active: %t, Busy: %t, UsageCount: %d}",
		c.ID, c.RemoteAddr, c.IsActive, c.IsBusy, c.UsageCount)
}