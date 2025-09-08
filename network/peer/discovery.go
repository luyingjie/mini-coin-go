package peer

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"
)

// Discovery 节点发现服务
type Discovery struct {
	manager     *Manager
	isRunning   bool
	stopChannel chan bool
}

// NewDiscovery 创建节点发现服务
func NewDiscovery(manager *Manager) *Discovery {
	return &Discovery{
		manager:     manager,
		stopChannel: make(chan bool),
	}
}

// Start 启动节点发现服务
func (d *Discovery) Start() {
	if d.isRunning {
		return
	}
	
	d.isRunning = true
	log.Println("启动节点发现服务")
	
	// 启动不同的发现策略
	go d.periodicDiscovery()
	go d.bootstrapDiscovery()
	go d.peerExchangeDiscovery()
}

// Stop 停止节点发现服务
func (d *Discovery) Stop() {
	if !d.isRunning {
		return
	}
	
	d.isRunning = false
	close(d.stopChannel)
	log.Println("停止节点发现服务")
}

// periodicDiscovery 定期节点发现
func (d *Discovery) periodicDiscovery() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			d.discoverNewPeers()
		case <-d.stopChannel:
			return
		}
	}
}

// bootstrapDiscovery 引导节点发现
func (d *Discovery) bootstrapDiscovery() {
	// 首次启动时立即尝试发现
	time.Sleep(5 * time.Second)
	d.connectToSeedNodes()
	
	// 然后定期重试
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			d.connectToSeedNodes()
		case <-d.stopChannel:
			return
		}
	}
}

// peerExchangeDiscovery 节点交换发现
func (d *Discovery) peerExchangeDiscovery() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			d.exchangePeersWithConnected()
		case <-d.stopChannel:
			return
		}
	}
}

// discoverNewPeers 发现新节点
func (d *Discovery) discoverNewPeers() {
	log.Println("开始发现新节点...")
	
	// 从已连接的节点请求更多节点信息
	connectedPeers := d.manager.GetConnectedPeers()
	for _, peer := range connectedPeers {
		go d.requestPeersFrom(peer)
	}
	
	// 尝试连接到评分较高的未连接节点
	bestPeers := d.manager.GetBestPeers(5)
	for _, peer := range bestPeers {
		if peer.GetStatus() == StatusDisconnected {
			go d.attemptConnection(peer)
		}
	}
}

// connectToSeedNodes 连接到种子节点
func (d *Discovery) connectToSeedNodes() {
	log.Println("尝试连接种子节点...")
	
	allPeers := d.manager.GetAllPeers()
	for _, peer := range allPeers {
		if peer.GetStatus() == StatusDisconnected {
			go d.attemptConnection(peer)
		}
	}
}

// exchangePeersWithConnected 与已连接节点交换节点信息
func (d *Discovery) exchangePeersWithConnected() {
	connectedPeers := d.manager.GetConnectedPeers()
	if len(connectedPeers) == 0 {
		return
	}
	
	// 随机选择一些已连接的节点进行节点信息交换
	exchangeCount := min(3, len(connectedPeers))
	selectedPeers := d.selectRandomPeers(connectedPeers, exchangeCount)
	
	for _, peer := range selectedPeers {
		go d.exchangePeersWith(peer)
	}
}

// selectRandomPeers 随机选择节点
func (d *Discovery) selectRandomPeers(peers []*Peer, count int) []*Peer {
	if len(peers) <= count {
		return peers
	}
	
	rand.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})
	
	return peers[:count]
}

// attemptConnection 尝试连接到节点
func (d *Discovery) attemptConnection(peer *Peer) {
	log.Printf("尝试连接节点: %s", peer.GetFullAddress())
	
	peer.UpdateStatus(StatusConnecting)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", peer.GetFullAddress())
	if err != nil {
		log.Printf("连接节点失败 %s: %v", peer.GetFullAddress(), err)
		peer.UpdateStatus(StatusFailed)
		return
	}
	
	conn.Close()
	peer.UpdateStatus(StatusConnected)
	log.Printf("成功连接节点: %s", peer.GetFullAddress())
	
	// 连接成功后，尝试获取该节点的节点列表
	go d.requestPeersFrom(peer)
}

// requestPeersFrom 从指定节点请求节点列表
func (d *Discovery) requestPeersFrom(peer *Peer) {
	// 这里应该发送节点列表请求消息
	// 由于这需要与network层集成，暂时模拟
	log.Printf("从节点 %s 请求节点列表", peer.GetFullAddress())
	
	// 模拟收到一些新节点地址
	d.simulateDiscoveredPeers()
}

// exchangePeersWith 与指定节点交换节点信息
func (d *Discovery) exchangePeersWith(peer *Peer) {
	log.Printf("与节点 %s 交换节点信息", peer.GetFullAddress())
	
	// 发送我们的节点列表给对方
	// 请求对方的节点列表
	// 这需要与network层集成实现
}

// simulateDiscoveredPeers 模拟发现的节点（用于测试）
func (d *Discovery) simulateDiscoveredPeers() {
	// 模拟一些发现的节点地址
	simulatedAddresses := []string{
		"127.0.0.1:3001",
		"127.0.0.1:3002",
		"127.0.0.1:3003",
	}
	
	for _, addr := range simulatedAddresses {
		if d.manager.GetPeer(addr) == nil {
			peer := NewPeerFromAddress(addr)
			if peer != nil {
				d.manager.AddPeer(peer)
				log.Printf("发现新节点: %s", addr)
			}
		}
	}
}

// DiscoverPeersFromBootstrap 从引导节点发现节点
func (d *Discovery) DiscoverPeersFromBootstrap(bootstrapNodes []string) {
	log.Println("从引导节点发现新节点...")
	
	for _, nodeAddr := range bootstrapNodes {
		peer := NewPeerFromAddress(nodeAddr)
		if peer != nil {
			d.manager.AddPeer(peer)
			go d.attemptConnection(peer)
		}
	}
}

// DiscoverPeersFromDNS 从DNS发现节点（预留接口）
func (d *Discovery) DiscoverPeersFromDNS(dnsSeeds []string) {
	log.Println("从DNS种子发现节点...")
	
	for _, seed := range dnsSeeds {
		go d.resolveDNSSeed(seed)
	}
}

// resolveDNSSeed 解析DNS种子
func (d *Discovery) resolveDNSSeed(seed string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, seed)
	if err != nil {
		log.Printf("DNS解析失败 %s: %v", seed, err)
		return
	}
	
	for _, ip := range ips {
		address := fmt.Sprintf("%s:3000", ip.IP.String())
		peer := NewPeerFromAddress(address)
		if peer != nil {
			d.manager.AddPeer(peer)
			go d.attemptConnection(peer)
		}
	}
}

// GetDiscoveryStats 获取发现统计信息
func (d *Discovery) GetDiscoveryStats() map[string]interface{} {
	return map[string]interface{}{
		"is_running":      d.isRunning,
		"connected_peers": len(d.manager.GetConnectedPeers()),
		"total_peers":     len(d.manager.GetAllPeers()),
	}
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}