package peer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Manager 节点管理器
type Manager struct {
	peers           map[string]*Peer // 所有节点
	maxPeers        int              // 最大节点数
	seedNodes       []string         // 种子节点
	mutex           sync.RWMutex     // 读写锁
	heartbeatTicker *time.Ticker     // 心跳定时器
	cleanupTicker   *time.Ticker     // 清理定时器
	configFile      string           // 配置文件路径
}

// PeerConfig 节点配置
type PeerConfig struct {
	SeedNodes []string `json:"seed_nodes"`
	MaxPeers  int      `json:"max_peers"`
}

// NewManager 创建节点管理器
func NewManager(configFile string) *Manager {
	manager := &Manager{
		peers:      make(map[string]*Peer),
		maxPeers:   50,
		configFile: configFile,
	}

	// 加载配置
	manager.loadConfig()

	// 初始化种子节点
	manager.initSeedNodes()

	// 启动心跳和清理任务
	manager.startBackgroundTasks()

	return manager
}

// loadConfig 加载配置文件
func (m *Manager) loadConfig() {
	data, err := ioutil.ReadFile(m.configFile)
	if err != nil {
		log.Printf("无法读取配置文件 %s: %v，使用默认配置", m.configFile, err)
		m.seedNodes = []string{"localhost:3000"}
		return
	}

	var config PeerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("配置文件格式错误: %v，使用默认配置", err)
		m.seedNodes = []string{"localhost:3000"}
		return
	}

	m.seedNodes = config.SeedNodes
	if config.MaxPeers > 0 {
		m.maxPeers = config.MaxPeers
	}

	log.Printf("加载配置成功：种子节点 %v，最大节点数 %d", m.seedNodes, m.maxPeers)
}

// initSeedNodes 初始化种子节点
func (m *Manager) initSeedNodes() {
	for _, seedAddr := range m.seedNodes {
		peer := NewPeerFromAddress(seedAddr)
		if peer != nil {
			m.AddPeer(peer)
		}
	}
}

// NewPeerFromAddress 从地址创建节点
func NewPeerFromAddress(address string) *Peer {
	host, port, err := parseAddress(address)
	if err != nil {
		log.Printf("地址解析失败 %s: %v", address, err)
		return nil
	}
	return NewPeer(host, port)
}

// parseAddress 解析地址
func parseAddress(address string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return "", 0, fmt.Errorf("地址格式无效: %s", address)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("端口格式无效: %s", portStr)
	}

	return host, port, nil
}

// AddPeer 添加节点
func (m *Manager) AddPeer(peer *Peer) {
	if peer == nil {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 检查是否已存在
	if existingPeer, exists := m.peers[peer.GetFullAddress()]; exists {
		// 更新现有节点信息
		existingPeer.UpdateLastSeen()
		return
	}

	// 检查节点数量限制
	if len(m.peers) >= m.maxPeers {
		// 移除评分最低的节点
		m.removeLowestScoredPeer()
	}

	m.peers[peer.GetFullAddress()] = peer
	log.Printf("添加新节点: %s", peer.String())
}

// removeLowestScoredPeer 移除评分最低的节点
func (m *Manager) removeLowestScoredPeer() {
	var lowestPeer *Peer
	lowestScore := int(^uint(0) >> 1) // 最大整数

	for _, peer := range m.peers {
		if peer.GetScore() < lowestScore && peer.GetStatus() != StatusConnected {
			lowestScore = peer.GetScore()
			lowestPeer = peer
		}
	}

	if lowestPeer != nil {
		delete(m.peers, lowestPeer.GetFullAddress())
		log.Printf("移除低评分节点: %s", lowestPeer.String())
	}
}

// GetPeer 获取指定地址的节点
func (m *Manager) GetPeer(address string) *Peer {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.peers[address]
}

// GetBestPeers 获取评分最高的节点列表
func (m *Manager) GetBestPeers(count int) []*Peer {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var peers []*Peer
	for _, peer := range m.peers {
		if peer.CanConnect() {
			peers = append(peers, peer)
		}
	}

	// 按评分排序
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].GetScore() > peers[j].GetScore()
	})

	if len(peers) > count {
		peers = peers[:count]
	}

	return peers
}

// GetRandomPeers 获取随机节点列表
func (m *Manager) GetRandomPeers(count int) []*Peer {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var availablePeers []*Peer
	for _, peer := range m.peers {
		if peer.CanConnect() {
			availablePeers = append(availablePeers, peer)
		}
	}

	if len(availablePeers) == 0 {
		return nil
	}

	// 随机打乱
	rand.Shuffle(len(availablePeers), func(i, j int) {
		availablePeers[i], availablePeers[j] = availablePeers[j], availablePeers[i]
	})

	if len(availablePeers) > count {
		availablePeers = availablePeers[:count]
	}

	return availablePeers
}

// GetAllPeers 获取所有节点
func (m *Manager) GetAllPeers() []*Peer {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var peers []*Peer
	for _, peer := range m.peers {
		peers = append(peers, peer)
	}
	return peers
}

// GetConnectedPeers 获取已连接的节点
func (m *Manager) GetConnectedPeers() []*Peer {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var peers []*Peer
	for _, peer := range m.peers {
		if peer.GetStatus() == StatusConnected {
			peers = append(peers, peer)
		}
	}
	return peers
}

// RemovePeer 移除节点
func (m *Manager) RemovePeer(address string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if peer, exists := m.peers[address]; exists {
		delete(m.peers, address)
		log.Printf("移除节点: %s", peer.String())
	}
}

// startBackgroundTasks 启动后台任务
func (m *Manager) startBackgroundTasks() {
	// 心跳检查（每30秒）
	m.heartbeatTicker = time.NewTicker(30 * time.Second)
	go m.heartbeatTask()

	// 清理任务（每5分钟）
	m.cleanupTicker = time.NewTicker(5 * time.Minute)
	go m.cleanupTask()
}

// heartbeatTask 心跳检查任务
func (m *Manager) heartbeatTask() {
	for range m.heartbeatTicker.C {
		m.performHeartbeat()
	}
}

// performHeartbeat 执行心跳检查
func (m *Manager) performHeartbeat() {
	peers := m.GetAllPeers()
	for _, peer := range peers {
		go func(p *Peer) {
			if err := p.Ping(); err != nil {
				log.Printf("节点心跳失败 %s: %v", p.GetFullAddress(), err)
			}
		}(peer)
	}
}

// cleanupTask 清理任务
func (m *Manager) cleanupTask() {
	for range m.cleanupTicker.C {
		m.performCleanup()
	}
}

// performCleanup 执行清理
func (m *Manager) performCleanup() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	timeout := 10 * time.Minute
	var toRemove []string

	for address, peer := range m.peers {
		if !peer.IsAlive(timeout) && peer.GetStatus() != StatusConnected {
			toRemove = append(toRemove, address)
		}
	}

	for _, address := range toRemove {
		delete(m.peers, address)
		log.Printf("清理死亡节点: %s", address)
	}

	log.Printf("清理完成，当前节点数: %d", len(m.peers))
}

// GetStats 获取统计信息
func (m *Manager) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_peers"] = len(m.peers)

	statusCount := make(map[string]int)
	for _, peer := range m.peers {
		statusCount[peer.GetStatus().String()]++
	}
	stats["status_count"] = statusCount
	stats["max_peers"] = m.maxPeers

	return stats
}

// Stop 停止管理器
func (m *Manager) Stop() {
	if m.heartbeatTicker != nil {
		m.heartbeatTicker.Stop()
	}
	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
	}
	log.Println("节点管理器已停止")
}
