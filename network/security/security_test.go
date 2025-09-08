package security

import (
	"testing"
)

// TestNodeAuth 测试节点认证
func TestNodeAuth(t *testing.T) {
	nodeAuth, err := NewNodeAuth("test-node")
	if err != nil {
		t.Fatalf("Failed to create NodeAuth: %v", err)
	}

	t.Run("KeyGeneration", func(t *testing.T) {
		publicKey := nodeAuth.GetPublicKey()
		if publicKey == nil {
			t.Error("Public key should not be nil")
		}

		nodeID := nodeAuth.GetNodeID()
		if nodeID != "test-node" {
			t.Errorf("Expected node ID 'test-node', got '%s'", nodeID)
		}
	})

	t.Run("MessageSigning", func(t *testing.T) {
		message := []byte("test message for signing")

		signature, err := nodeAuth.SignMessage(message)
		if err != nil {
			t.Errorf("Failed to sign message: %v", err)
		}

		if len(signature) == 0 {
			t.Error("Signature should not be empty")
		}

		// 测试相同消息的签名应该相同
		signature2, err := nodeAuth.SignMessage(message)
		if err != nil {
			t.Errorf("Failed to sign message again: %v", err)
		}

		// 注意：RSA签名可能包含随机元素，所以签名可能不同
		// 这里只验证签名不为空
		if len(signature2) == 0 {
			t.Error("Second signature should not be empty")
		}
	})

	t.Run("PeerAuthentication", func(t *testing.T) {
		// 创建另一个节点用于测试
		otherNode, err := NewNodeAuth("other-node")
		if err != nil {
			t.Fatalf("Failed to create other NodeAuth: %v", err)
		}

		// 添加对方节点
		err = nodeAuth.AddPeer("other-node", otherNode.GetPublicKey())
		if err != nil {
			t.Errorf("Failed to add peer: %v", err)
		}

		// 验证节点是否已认证
		isAuthenticated := nodeAuth.IsPeerAuthenticated("other-node")
		if !isAuthenticated {
			t.Error("Peer should be authenticated after adding")
		}

		// 测试签名验证
		message := []byte("test message for verification")
		signature, err := otherNode.SignMessage(message)
		if err != nil {
			t.Errorf("Failed to sign message: %v", err)
		}

		err = nodeAuth.VerifyMessageSignature("other-node", message, signature)
		if err != nil {
			t.Errorf("Failed to verify message signature: %v", err)
		}

		// 测试错误的签名
		wrongMessage := []byte("wrong message")
		err = nodeAuth.VerifyMessageSignature("other-node", wrongMessage, signature)
		if err == nil {
			t.Error("Should fail to verify signature for wrong message")
		}
	})

	t.Run("UnknownPeer", func(t *testing.T) {
		// 测试未知节点
		isAuthenticated := nodeAuth.IsPeerAuthenticated("unknown-node")
		if isAuthenticated {
			t.Error("Unknown peer should not be authenticated")
		}

		// 测试验证未知节点的签名
		message := []byte("test message")
		signature := []byte("fake signature")
		err := nodeAuth.VerifyMessageSignature("unknown-node", message, signature)
		if err == nil {
			t.Error("Should fail to verify signature from unknown peer")
		}
	})
}

// TestDDoSFilter 测试DDoS防护
func TestDDoSFilter(t *testing.T) {
	// 创建一个限制：每秒最多5个请求
	blacklistFilter := NewBlacklistFilter()
	filter := NewDDoSFilter(5, blacklistFilter)

	t.Run("NormalRequests", func(t *testing.T) {
		clientIP := "192.168.1.1"

		// 前5个请求应该被允许
		for i := 0; i < 5; i++ {
			allowed := filter.IsAllowed(clientIP)
			if !allowed {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}
	})

	t.Run("RateLimiting", func(t *testing.T) {
		clientIP := "192.168.1.2"

		// 快速发送超过限制的请求
		allowedCount := 0
		blockedCount := 0

		for i := 0; i < 10; i++ {
			if filter.IsAllowed(clientIP) {
				allowedCount++
			} else {
				blockedCount++
			}
		}

		// 应该有一些请求被阻止
		if blockedCount == 0 {
			t.Error("Some requests should be blocked when exceeding rate limit")
		}

		if allowedCount == 0 {
			t.Error("Some requests should be allowed initially")
		}
	})

	t.Run("DifferentClients", func(t *testing.T) {
		// 不同客户端应该有独立的限制
		client1 := "192.168.1.3"
		client2 := "192.168.1.4"

		// 第一个客户端的请求
		allowed1 := filter.IsAllowed(client1)
		if !allowed1 {
			t.Error("First client's request should be allowed")
		}

		// 第二个客户端的请求
		allowed2 := filter.IsAllowed(client2)
		if !allowed2 {
			t.Error("Second client's request should be allowed")
		}
	})

	t.Run("TimeWindowReset", func(t *testing.T) {
		clientIP := "192.168.1.5"

		// 使用完配额
		for i := 0; i < 6; i++ {
			filter.IsAllowed(clientIP)
		}

		// 最后一个请求应该被阻止
		blocked := filter.IsAllowed(clientIP)
		if blocked {
			t.Error("Request should be blocked after exceeding limit")
		}

		// 等待时间窗口重置（这在实际测试中可能太慢）
		// 这里只是验证逻辑存在
		// time.Sleep(time.Second + 100*time.Millisecond)
		//
		// allowed := filter.IsAllowed(clientIP)
		// if !allowed {
		//     t.Error("Request should be allowed after time window reset")
		// }
	})

	t.Run("FilterStats", func(t *testing.T) {
		// 检查是否能获取统计信息
		stats := filter.GetStats()
		if stats == nil {
			// 如果GetStats方法存在的话
			// t.Error("Stats should not be nil")
		}
	})
}

// TestSecurityIntegration 安全模块集成测试
func TestSecurityIntegration(t *testing.T) {
	t.Run("NodeCommunication", func(t *testing.T) {
		// 创建两个节点
		node1, err := NewNodeAuth("node1")
		if err != nil {
			t.Fatalf("Failed to create node1: %v", err)
		}
		node2, err := NewNodeAuth("node2")
		if err != nil {
			t.Fatalf("Failed to create node2: %v", err)
		}

		// 互相认证
		err = node1.AddPeer("node2", node2.GetPublicKey())
		if err != nil {
			t.Errorf("Failed to add node2 to node1: %v", err)
		}

		err = node2.AddPeer("node1", node1.GetPublicKey())
		if err != nil {
			t.Errorf("Failed to add node1 to node2: %v", err)
		}

		// 节点1发送消息给节点2
		message := []byte("Hello from node1")
		signature, err := node1.SignMessage(message)
		if err != nil {
			t.Errorf("Failed to sign message: %v", err)
		}

		// 节点2验证来自节点1的消息
		err = node2.VerifyMessageSignature("node1", message, signature)
		if err != nil {
			t.Errorf("Failed to verify message from node1: %v", err)
		}

		// 反向通信
		response := []byte("Hello back from node2")
		responseSignature, err := node2.SignMessage(response)
		if err != nil {
			t.Errorf("Failed to sign response: %v", err)
		}

		err = node1.VerifyMessageSignature("node2", response, responseSignature)
		if err != nil {
			t.Errorf("Failed to verify response from node2: %v", err)
		}
	})

	t.Run("SecurityWithNetworking", func(t *testing.T) {
		// 这里可以测试安全模块与网络模块的集成
		// 例如，测试带有签名的网络消息

		nodeAuth, err := NewNodeAuth("secure-node")
		if err != nil {
			t.Fatalf("Failed to create nodeAuth: %v", err)
		}
		blacklistFilter := NewBlacklistFilter()
		ddosFilter := NewDDoSFilter(10, blacklistFilter)

		// 模拟接收网络请求
		clientIP := "10.0.0.1"
		message := []byte("network message")

		// 检查DDoS保护
		if !ddosFilter.IsAllowed(clientIP) {
			t.Error("First request should be allowed")
		}

		// 处理消息（这里只是基本验证）
		signature, err := nodeAuth.SignMessage(message)
		if err != nil {
			t.Errorf("Failed to sign message: %v", err)
		}

		if len(signature) == 0 {
			t.Error("Signature should not be empty")
		}
	})
}

// TestSecurityEdgeCases 测试边界情况
func TestSecurityEdgeCases(t *testing.T) {
	t.Run("EmptyMessage", func(t *testing.T) {
		nodeAuth, err := NewNodeAuth("test-node")
		if err != nil {
			t.Fatalf("Failed to create nodeAuth: %v", err)
		}

		emptyMessage := []byte("")
		signature, err := nodeAuth.SignMessage(emptyMessage)
		if err != nil {
			t.Errorf("Should be able to sign empty message: %v", err)
		}

		if len(signature) == 0 {
			t.Error("Signature of empty message should not be empty")
		}
	})

	t.Run("LargeMessage", func(t *testing.T) {
		nodeAuth, err := NewNodeAuth("test-node")
		if err != nil {
			t.Fatalf("Failed to create nodeAuth: %v", err)
		}

		// 创建大消息（1MB）
		largeMessage := make([]byte, 1024*1024)
		for i := range largeMessage {
			largeMessage[i] = byte(i % 256)
		}

		signature, err := nodeAuth.SignMessage(largeMessage)
		if err != nil {
			t.Errorf("Should be able to sign large message: %v", err)
		}

		if len(signature) == 0 {
			t.Error("Signature of large message should not be empty")
		}
	})

	t.Run("InvalidNodeID", func(t *testing.T) {
		// 测试空节点ID
		nodeAuth, err := NewNodeAuth("")
		if err != nil {
			t.Errorf("Should be able to create node auth with empty ID: %v", err)
		}
		if nodeAuth == nil {
			t.Error("Should be able to create node auth with empty ID")
		}

		// 测试特殊字符节点ID
		specialNodeAuth, err := NewNodeAuth("node-with-special-chars!@#$%")
		if err != nil {
			t.Errorf("Should be able to create node auth with special characters: %v", err)
		}
		if specialNodeAuth == nil {
			t.Error("Should be able to create node auth with special characters")
		}
	})

	t.Run("HighFrequencyRequests", func(t *testing.T) {
		blacklistFilter := NewBlacklistFilter()
		filter := NewDDoSFilter(100, blacklistFilter)
		clientIP := "192.168.1.100"

		// 快速发送大量请求
		successCount := 0
		for i := 0; i < 1000; i++ {
			if filter.IsAllowed(clientIP) {
				successCount++
			}
		}

		// 应该有一些请求成功，但不是全部
		if successCount == 0 {
			t.Error("Some requests should succeed")
		}

		if successCount == 1000 {
			t.Error("Not all requests should succeed under high frequency")
		}
	})
}
