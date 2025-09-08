package connection

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// PoolConfig 连接池配置
type PoolConfig struct {
	MaxConnections   int           // 最大连接数
	MaxIdleTime      time.Duration // 最大空闲时间
	ConnectTimeout   time.Duration // 连接超时时间
	HealthCheckInterval time.Duration // 健康检查间隔
	RetryInterval    time.Duration // 重试间隔
	MaxRetries       int           // 最大重试次数
}

// DefaultPoolConfig 默认连接池配置
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxConnections:      20,
		MaxIdleTime:         5 * time.Minute,
		ConnectTimeout:      10 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		RetryInterval:       5 * time.Second,
		MaxRetries:          3,
	}
}

// Pool 连接池
type Pool struct {
	address      string                    // 目标地址
	connections  map[string]*Connection    // 连接映射
	available    chan *Connection          // 可用连接队列
	config       *PoolConfig               // 配置
	mutex        sync.RWMutex              // 读写锁
	isRunning    bool                      // 是否运行中
	stopCh       chan bool                 // 停止信号
	healthTicker *time.Ticker              // 健康检查定时器
	stats        *PoolStats                // 统计信息
}

// PoolStats 连接池统计信息
type PoolStats struct {
	TotalConnections   int           // 总连接数
	ActiveConnections  int           // 活跃连接数
	IdleConnections    int           // 空闲连接数
	FailedConnections  int           // 失败连接数
	TotalRequests      int64         // 总请求数
	SuccessfulRequests int64         // 成功请求数
	AverageResponseTime time.Duration // 平均响应时间
	mutex              sync.RWMutex   // 读写锁
}

// NewPool 创建新的连接池
func NewPool(address string, config *PoolConfig) *Pool {
	if config == nil {
		config = DefaultPoolConfig()
	}
	
	pool := &Pool{
		address:     address,
		connections: make(map[string]*Connection),
		available:   make(chan *Connection, config.MaxConnections),
		config:      config,
		isRunning:   false,
		stopCh:      make(chan bool),
		stats:       &PoolStats{},
	}
	
	return pool
}

// Start 启动连接池
func (p *Pool) Start() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if p.isRunning {
		return fmt.Errorf("连接池已经在运行")
	}
	
	p.isRunning = true
	log.Printf("启动连接池，目标地址: %s", p.address)
	
	// 预创建一些连接
	go p.preCreateConnections()
	
	// 启动健康检查
	p.startHealthCheck()
	
	return nil
}

// Stop 停止连接池
func (p *Pool) Stop() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if !p.isRunning {
		return nil
	}
	
	p.isRunning = false
	close(p.stopCh)
	
	// 停止健康检查
	if p.healthTicker != nil {
		p.healthTicker.Stop()
	}
	
	// 关闭所有连接
	for _, conn := range p.connections {
		conn.Close()
	}
	
	// 清空可用连接队列
	for len(p.available) > 0 {
		<-p.available
	}
	
	log.Printf("连接池已停止，地址: %s", p.address)
	return nil
}

// GetConnection 获取连接
func (p *Pool) GetConnection(ctx context.Context) (*Connection, error) {
	if !p.isRunning {
		return nil, fmt.Errorf("连接池未运行")
	}
	
	p.stats.mutex.Lock()
	p.stats.TotalRequests++
	p.stats.mutex.Unlock()
	
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		p.updateAverageResponseTime(duration)
	}()
	
	select {
	case conn := <-p.available:
		// 检查连接健康状态
		if conn.IsHealthy() {
			conn.Use()
			p.stats.mutex.Lock()
			p.stats.SuccessfulRequests++
			p.stats.mutex.Unlock()
			return conn, nil
		}
		// 连接不健康，关闭并创建新连接
		conn.Close()
		p.removeConnection(conn.ID)
		
	case <-ctx.Done():
		return nil, ctx.Err()
		
	default:
		// 没有可用连接，尝试创建新连接
	}
	
	// 创建新连接
	return p.createConnection(ctx)
}

// ReturnConnection 归还连接
func (p *Pool) ReturnConnection(conn *Connection) {
	if conn == nil {
		return
	}
	
	conn.Release()
	
	// 如果连接仍然健康且池未满，则归还到池中
	if conn.IsHealthy() && len(p.available) < p.config.MaxConnections {
		select {
		case p.available <- conn:
			// 成功归还
		default:
			// 池已满，关闭连接
			conn.Close()
			p.removeConnection(conn.ID)
		}
	} else {
		// 连接不健康或池已满，关闭连接
		conn.Close()
		p.removeConnection(conn.ID)
	}
}

// preCreateConnections 预创建连接
func (p *Pool) preCreateConnections() {
	initialCount := min(3, p.config.MaxConnections/2)
	
	for i := 0; i < initialCount; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), p.config.ConnectTimeout)
		conn, err := p.createConnection(ctx)
		cancel()
		
		if err != nil {
			log.Printf("预创建连接失败: %v", err)
			continue
		}
		
		p.ReturnConnection(conn)
	}
}

// createConnection 创建新连接
func (p *Pool) createConnection(ctx context.Context) (*Connection, error) {
	// 检查连接数限制
	p.mutex.RLock()
	currentCount := len(p.connections)
	p.mutex.RUnlock()
	
	if currentCount >= p.config.MaxConnections {
		return nil, fmt.Errorf("连接数已达上限: %d", p.config.MaxConnections)
	}
	
	// 创建网络连接
	var dialer net.Dialer
	netConn, err := dialer.DialContext(ctx, "tcp", p.address)
	if err != nil {
		p.stats.mutex.Lock()
		p.stats.FailedConnections++
		p.stats.mutex.Unlock()
		return nil, fmt.Errorf("连接失败: %v", err)
	}
	
	conn := NewConnection(netConn)
	
	// 添加到连接映射
	p.mutex.Lock()
	p.connections[conn.ID] = conn
	p.mutex.Unlock()
	
	log.Printf("创建新连接: %s -> %s", conn.ID, p.address)
	return conn, nil
}

// removeConnection 移除连接
func (p *Pool) removeConnection(connID string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if conn, exists := p.connections[connID]; exists {
		delete(p.connections, connID)
		log.Printf("移除连接: %s", conn.ID)
	}
}

// startHealthCheck 启动健康检查
func (p *Pool) startHealthCheck() {
	p.healthTicker = time.NewTicker(p.config.HealthCheckInterval)
	go p.healthCheckTask()
}

// healthCheckTask 健康检查任务
func (p *Pool) healthCheckTask() {
	for {
		select {
		case <-p.healthTicker.C:
			p.performHealthCheck()
		case <-p.stopCh:
			return
		}
	}
}

// performHealthCheck 执行健康检查
func (p *Pool) performHealthCheck() {
	p.mutex.RLock()
	connections := make([]*Connection, 0, len(p.connections))
	for _, conn := range p.connections {
		connections = append(connections, conn)
	}
	p.mutex.RUnlock()
	
	var unhealthyConns []*Connection
	
	for _, conn := range connections {
		if !conn.IsHealthy() || conn.IsExpired(p.config.MaxIdleTime) {
			unhealthyConns = append(unhealthyConns, conn)
		}
	}
	
	// 关闭不健康的连接
	for _, conn := range unhealthyConns {
		conn.Close()
		p.removeConnection(conn.ID)
	}
	
	if len(unhealthyConns) > 0 {
		log.Printf("健康检查: 移除 %d 个不健康连接", len(unhealthyConns))
	}
}

// updateAverageResponseTime 更新平均响应时间
func (p *Pool) updateAverageResponseTime(duration time.Duration) {
	p.stats.mutex.Lock()
	defer p.stats.mutex.Unlock()
	
	if p.stats.AverageResponseTime == 0 {
		p.stats.AverageResponseTime = duration
	} else {
		// 简单的移动平均
		p.stats.AverageResponseTime = (p.stats.AverageResponseTime + duration) / 2
	}
}

// GetStats 获取连接池统计信息
func (p *Pool) GetStats() *PoolStats {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	p.stats.mutex.Lock()
	defer p.stats.mutex.Unlock()
	
	activeCount := 0
	idleCount := 0
	
	for _, conn := range p.connections {
		if conn.IsBusy {
			activeCount++
		} else {
			idleCount++
		}
	}
	
	return &PoolStats{
		TotalConnections:    len(p.connections),
		ActiveConnections:   activeCount,
		IdleConnections:     idleCount,
		FailedConnections:   p.stats.FailedConnections,
		TotalRequests:       p.stats.TotalRequests,
		SuccessfulRequests:  p.stats.SuccessfulRequests,
		AverageResponseTime: p.stats.AverageResponseTime,
	}
}

// GetConnectionInfo 获取所有连接信息
func (p *Pool) GetConnectionInfo() []map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	var info []map[string]interface{}
	for _, conn := range p.connections {
		info = append(info, conn.GetInfo())
	}
	
	return info
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}