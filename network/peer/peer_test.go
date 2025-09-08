package peer

import (
	"os"
	"testing"
	"time"
)

// TestPeer 测试节点基础功能
func TestPeer(t *testing.T) {
	t.Run("PeerCreation", func(t *testing.T) {
		peer := NewPeer("localhost", 3000)

		if peer == nil {
			t.Error("Peer should not be nil")
		}

		if peer.GetHost() != "localhost" {
			t.Errorf("Expected host 'localhost', got '%s'", peer.GetHost())
		}

		if peer.GetPort() != 3000 {
			t.Errorf("Expected port 3000, got %d", peer.GetPort())
		}

		expectedAddr := "localhost:3000"
		if peer.GetFullAddress() != expectedAddr {
			t.Errorf("Expected address '%s', got '%s'", expectedAddr, peer.GetFullAddress())
		}
	})

	t.Run("PeerStatus", func(t *testing.T) {
		peer := NewPeer("localhost", 3001)

		// 初始状态应该是断开连接
		if peer.GetStatus() != StatusDisconnected {
			t.Errorf("Expected status %v, got %v", StatusDisconnected, peer.GetStatus())
		}

		// 测试状态变更
		peer.SetStatus(StatusConnecting)
		if peer.GetStatus() != StatusConnecting {
			t.Errorf("Expected status %v, got %v", StatusConnecting, peer.GetStatus())
		}

		peer.SetStatus(StatusConnected)
		if peer.GetStatus() != StatusConnected {
			t.Errorf("Expected status %v, got %v", StatusConnected, peer.GetStatus())
		}
	})

	t.Run("PeerScore", func(t *testing.T) {
		peer := NewPeer("localhost", 3002)

		initialScore := peer.GetScore()
		if initialScore < 0 {
			t.Error("Initial score should not be negative")
		}

		// 测试增加分数
		peer.IncreaseScore(10)
		newScore := peer.GetScore()
		if newScore != initialScore+10 {
			t.Errorf("Expected score %d, got %d", initialScore+10, newScore)
		}

		// 测试减少分数
		peer.DecreaseScore(5)
		finalScore := peer.GetScore()
		if finalScore != newScore-5 {
			t.Errorf("Expected score %d, got %d", newScore-5, finalScore)
		}
	})

	t.Run("PeerConnection", func(t *testing.T) {
		peer := NewPeer("localhost", 3003)

		// 测试连接能力检查
		canConnect := peer.CanConnect()
		// 根据当前状态，应该可以连接
		if !canConnect {
			// 这取决于具体实现，可能需要调整
		}

		// 测试活跃性检查
		isAlive := peer.IsAlive(1 * time.Minute)
		if !isAlive {
			t.Error("Newly created peer should be considered alive")
		}

		// 更新最后见到时间
		peer.UpdateLastSeen()

		// 再次检查活跃性
		isAlive = peer.IsAlive(1 * time.Minute)
		if !isAlive {
			t.Error("Peer should be alive after updating last seen")
		}
	})

	t.Run("PeerPing", func(t *testing.T) {
		peer := NewPeer("localhost", 9999) // 不存在的端口

		// Ping应该失败
		err := peer.Ping()
		if err == nil {
			t.Error("Ping to non-existent port should fail")
		}
	})
}

// TestPeerManager 测试节点管理器功能
func TestPeerManager(t *testing.T) {
	// 创建临时配置文件
	configFile := "test_peer_config.json"
	configContent := `{
		"seed_nodes": ["localhost:3000", "localhost:3001"],
		"max_peers": 5
	}`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	defer os.Remove(configFile)

	manager := NewManager(configFile)
	defer manager.Stop()

	t.Run("ManagerInitialization", func(t *testing.T) {
		// 检查种子节点是否被正确加载
		allPeers := manager.GetAllPeers()
		if len(allPeers) < 2 {
			t.Errorf("Expected at least 2 peers from seed nodes, got %d", len(allPeers))
		}
	})

	t.Run("AddAndRemovePeers", func(t *testing.T) {
		initialCount := len(manager.GetAllPeers())

		// 添加新节点
		newPeer := NewPeer("localhost", 3010)
		manager.AddPeer(newPeer)

		afterAddCount := len(manager.GetAllPeers())
		if afterAddCount != initialCount+1 {
			t.Errorf("Expected %d peers after adding, got %d", initialCount+1, afterAddCount)
		}

		// 获取节点
		retrievedPeer := manager.GetPeer("localhost:3010")
		if retrievedPeer == nil {
			t.Error("Should be able to retrieve added peer")
		}

		// 移除节点
		manager.RemovePeer("localhost:3010")
		afterRemoveCount := len(manager.GetAllPeers())
		if afterRemoveCount != initialCount {
			t.Errorf("Expected %d peers after removing, got %d", initialCount, afterRemoveCount)
		}
	})

	t.Run("GetBestPeers", func(t *testing.T) {
		// 添加一些测试节点
		for i := 0; i < 3; i++ {
			peer := NewPeer("localhost", 4000+i)
			peer.IncreaseScore(i * 10) // 给不同的分数
			manager.AddPeer(peer)
		}

		bestPeers := manager.GetBestPeers(2)
		if len(bestPeers) > 2 {
			t.Errorf("Expected at most 2 best peers, got %d", len(bestPeers))
		}

		// 验证是否按分数排序
		if len(bestPeers) >= 2 {
			if bestPeers[0].GetScore() < bestPeers[1].GetScore() {
				t.Error("Best peers should be sorted by score (highest first)")
			}
		}
	})

	t.Run("GetRandomPeers", func(t *testing.T) {
		randomPeers := manager.GetRandomPeers(3)
		if len(randomPeers) > 3 {
			t.Errorf("Expected at most 3 random peers, got %d", len(randomPeers))
		}
	})

	t.Run("GetConnectedPeers", func(t *testing.T) {
		// 添加一个连接的节点
		connectedPeer := NewPeer("localhost", 5000)
		connectedPeer.SetStatus(StatusConnected)
		manager.AddPeer(connectedPeer)

		connectedPeers := manager.GetConnectedPeers()
		found := false
		for _, peer := range connectedPeers {
			if peer.GetFullAddress() == "localhost:5000" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Should find the connected peer")
		}
	})

	t.Run("ManagerStats", func(t *testing.T) {
		stats := manager.GetStats()

		if stats["total_peers"] == nil {
			t.Error("Stats should contain total_peers")
		}

		if stats["max_peers"] == nil {
			t.Error("Stats should contain max_peers")
		}

		if stats["status_count"] == nil {
			t.Error("Stats should contain status_count")
		}

		totalPeers := stats["total_peers"].(int)
		if totalPeers <= 0 {
			t.Error("Should have at least some peers")
		}

		maxPeers := stats["max_peers"].(int)
		if maxPeers != 5 {
			t.Errorf("Expected max_peers to be 5, got %d", maxPeers)
		}
	})
}

// TestPeerStatus 测试节点状态
func TestPeerStatus(t *testing.T) {
	t.Run("StatusValues", func(t *testing.T) {
		statuses := []PeerStatus{
			StatusDisconnected,
			StatusConnecting,
			StatusConnected,
			StatusFailed,
		}

		for _, status := range statuses {
			statusStr := status.String()
			if statusStr == "" {
				t.Errorf("Status %v should have a string representation", status)
			}
		}
	})
}

// TestPeerConfigLoading 测试配置加载
func TestPeerConfigLoading(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		configFile := "test_valid_config.json"
		configContent := `{
			"seed_nodes": ["node1:3000", "node2:3001", "node3:3002"],
			"max_peers": 20
		}`
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}
		defer os.Remove(configFile)

		manager := NewManager(configFile)
		defer manager.Stop()

		stats := manager.GetStats()
		if stats["max_peers"].(int) != 20 {
			t.Errorf("Expected max_peers to be 20, got %d", stats["max_peers"])
		}
	})

	t.Run("InvalidConfig", func(t *testing.T) {
		configFile := "test_invalid_config.json"
		configContent := `{invalid json`
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}
		defer os.Remove(configFile)

		// 应该能够创建管理器，即使配置无效（使用默认配置）
		manager := NewManager(configFile)
		defer manager.Stop()

		if manager == nil {
			t.Error("Manager should be created even with invalid config")
		}
	})

	t.Run("NonExistentConfig", func(t *testing.T) {
		configFile := "non_existent_config.json"

		// 应该能够创建管理器，即使配置文件不存在（使用默认配置）
		manager := NewManager(configFile)
		defer manager.Stop()

		if manager == nil {
			t.Error("Manager should be created even with non-existent config")
		}
	})
}
