package peer

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"
)

// PeerStatus 节点状态枚举
type PeerStatus int

const (
	StatusDisconnected PeerStatus = iota
	StatusConnecting
	StatusConnected
	StatusFailed
)

// String returns the string representation of PeerStatus
func (s PeerStatus) String() string {
	switch s {
	case StatusDisconnected:
		return "disconnected"
	case StatusConnecting:
		return "connecting"
	case StatusConnected:
		return "connected"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Peer 表示网络中的一个节点
type Peer struct {
	ID              string        // 节点唯一标识
	Address         string        // 节点地址
	Port            int           // 节点端口
	Status          PeerStatus    // 节点状态
	LastSeen        time.Time     // 最后活跃时间
	ConnectedAt     time.Time     // 连接建立时间
	Score           int           // 节点评分
	FailedAttempts  int           // 连续失败次数
	Version         int           // 协议版本
	BestHeight      int           // 最佳区块高度
	Services        uint64        // 支持的服务
	UserAgent       string        // 用户代理信息
	PingTime        time.Duration // 延迟时间
	mutex           sync.RWMutex  // 读写锁
}

// NewPeer 创建新的节点实例
func NewPeer(address string, port int) *Peer {
	return &Peer{
		ID:             generatePeerID(),
		Address:        address,
		Port:           port,
		Status:         StatusDisconnected,
		LastSeen:       time.Now(),
		Score:          50, // 默认评分
		FailedAttempts: 0,
		Version:        1,
		Services:       1, // 基础服务
		UserAgent:      "mini-coin-go/1.0",
	}
}

// generatePeerID 生成节点唯一标识
func generatePeerID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetFullAddress 获取完整地址
func (p *Peer) GetFullAddress() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return fmt.Sprintf("%s:%d", p.Address, p.Port)
}

// UpdateStatus 更新节点状态
func (p *Peer) UpdateStatus(status PeerStatus) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	oldStatus := p.Status
	p.Status = status
	p.LastSeen = time.Now()
	
	if status == StatusConnected {
		p.ConnectedAt = time.Now()
		p.FailedAttempts = 0
		p.Score += 10 // 连接成功增加评分
	} else if status == StatusFailed {
		p.FailedAttempts++
		p.Score -= 5 // 连接失败减少评分
		if p.Score < 0 {
			p.Score = 0
		}
	}
	
	fmt.Printf("节点 %s 状态变更: %s -> %s\n", p.ID, oldStatus, status)
}

// UpdateLastSeen 更新最后活跃时间
func (p *Peer) UpdateLastSeen() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.LastSeen = time.Now()
}

// UpdateBestHeight 更新最佳区块高度
func (p *Peer) UpdateBestHeight(height int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.BestHeight = height
}

// UpdatePingTime 更新延迟时间
func (p *Peer) UpdatePingTime(duration time.Duration) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.PingTime = duration
}

// IsAlive 判断节点是否活跃
func (p *Peer) IsAlive(timeout time.Duration) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return time.Since(p.LastSeen) < timeout
}

// CanConnect 判断是否可以连接
func (p *Peer) CanConnect() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	// 如果连续失败次数过多，需要等待一段时间
	if p.FailedAttempts > 5 {
		return time.Since(p.LastSeen) > time.Minute*5
	}
	
	return p.Status == StatusDisconnected
}

// GetScore 获取节点评分
func (p *Peer) GetScore() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.Score
}

// GetStatus 获取节点状态
func (p *Peer) GetStatus() PeerStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.Status
}

// String 返回节点的字符串表示
func (p *Peer) String() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	return fmt.Sprintf("Peer{ID: %s, Address: %s, Status: %s, Score: %d, Height: %d}",
		p.ID, p.GetFullAddress(), p.Status, p.Score, p.BestHeight)
}

// Ping 测试节点连接延迟
func (p *Peer) Ping() error {
	start := time.Now()
	
	conn, err := net.DialTimeout("tcp", p.GetFullAddress(), time.Second*5)
	if err != nil {
		p.UpdateStatus(StatusFailed)
		return fmt.Errorf("ping失败: %v", err)
	}
	defer conn.Close()
	
	duration := time.Since(start)
	p.UpdatePingTime(duration)
	p.UpdateLastSeen()
	
	return nil
}