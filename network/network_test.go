package network

import (
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"mini-coin-go/blockchain"
	"mini-coin-go/network/connection"
	"mini-coin-go/network/message"
	"mini-coin-go/network/peer"
	"mini-coin-go/network/security"
)

const (
	testNodeID = "test_network"
)

// setupNetworkTestEnvironment 设置网络测试环境
func setupNetworkTestEnvironment() {
	// 设置环境变量
	os.Setenv("NODE_ID", testNodeID)

	// 清理测试文件
	cleanupTestFiles()
}

// teardownNetworkTestEnvironment 清理网络测试环境
func teardownNetworkTestEnvironment() {
	os.Unsetenv("NODE_ID")
	cleanupTestFiles()
}

// cleanupTestFiles 清理测试文件
func cleanupTestFiles() {
	files := []string{
		"blockchain_test_network.db",
		"wallet_test_network.dat",
		"chainstate_test_network.db",
	}

	for _, file := range files {
		os.Remove(file)
	}
}

// TestNetworkProtocol 测试网络协议基础功能
func TestNetworkProtocol(t *testing.T) {
	setupNetworkTestEnvironment()
	defer teardownNetworkTestEnvironment()

	// 测试命令转换
	t.Run("CommandConversion", func(t *testing.T) {
		command := "version"
		bytes := CommandToBytes(command)

		if len(bytes) != commandLength {
			t.Errorf("Expected command length %d, got %d", commandLength, len(bytes))
		}

		converted := BytesToCommand(bytes)
		if converted != command {
			t.Errorf("Expected command %s, got %s", command, converted)
		}
	})

	// 测试数据编码
	t.Run("DataEncoding", func(t *testing.T) {
		data := Version{
			Version:    1,
			BestHeight: 100,
			AddrFrom:   "localhost:3000",
		}

		encoded, err := GobEncode(data)
		if err != nil {
			t.Errorf("Failed to encode data: %v", err)
		}

		if len(encoded) == 0 {
			t.Error("Encoded data should not be empty")
		}
	})
}

// TestConnectionManager 测试连接管理器
func TestConnectionManager(t *testing.T) {
	setupNetworkTestEnvironment()
	defer teardownNetworkTestEnvironment()

	config := connection.DefaultPoolConfig()
	config.MaxConnections = 2
	config.MaxIdleTime = 1 * time.Second

	manager := connection.NewManager(config)

	t.Run("ManagerLifecycle", func(t *testing.T) {
		// 启动管理器
		err := manager.Start()
		if err != nil {
			t.Errorf("Failed to start manager: %v", err)
		}

		// 检查状态
		stats := manager.GetManagerStats()
		if !stats["is_running"].(bool) {
			t.Error("Manager should be running")
		}

		// 停止管理器
		err = manager.Stop()
		if err != nil {
			t.Errorf("Failed to stop manager: %v", err)
		}
	})

	t.Run("ConnectionOperations", func(t *testing.T) {
		err := manager.Start()
		if err != nil {
			t.Errorf("Failed to start manager: %v", err)
		}
		defer manager.Stop()

		// 测试Ping功能
		err = manager.Ping("invalid-address:9999")
		if err == nil {
			t.Error("Ping to invalid address should fail")
		}

		// 测试获取活跃地址
		addresses := manager.GetActiveAddresses()
		if len(addresses) != 0 {
			t.Errorf("Expected 0 active addresses, got %d", len(addresses))
		}
	})
}

// TestMessageHandler 测试消息处理器
func TestMessageHandler(t *testing.T) {
	setupNetworkTestEnvironment()
	defer teardownNetworkTestEnvironment()

	t.Run("HandlerLifecycle", func(t *testing.T) {
		handler := message.NewHandler(2)

		// 启动处理器
		err := handler.Start()
		if err != nil {
			t.Errorf("Failed to start handler: %v", err)
		}

		// 检查状态
		if !handler.IsRunning() {
			t.Error("Handler should be running")
		}

		// 停止处理器
		err = handler.Stop()
		if err != nil {
			t.Errorf("Failed to stop handler: %v", err)
		}
	})

	t.Run("MessageProcessing", func(t *testing.T) {
		handler := message.NewHandler(2)
		var processedCount int
		var mutex sync.Mutex

		// 注册处理函数
		handler.RegisterHandler("test", func(msg *message.Message) error {
			mutex.Lock()
			processedCount++
			mutex.Unlock()
			return nil
		})

		err := handler.Start()
		if err != nil {
			t.Errorf("Failed to start handler: %v", err)
		}

		// 提交测试消息
		testMsg := &message.Message{
			ID:         "test-msg-1",
			Type:       "test",
			Payload:    []byte("test data"),
			Priority:   message.PriorityNormal,
			Timeout:    5 * time.Second,
			CreatedAt:  time.Now(),
			MaxRetries: 3,
		}

		err = handler.Submit(testMsg)
		if err != nil {
			t.Errorf("Failed to submit message: %v", err)
		}

		// 等待处理完成
		time.Sleep(1 * time.Second)

		mutex.Lock()
		count := processedCount
		mutex.Unlock()

		if count != 1 {
			t.Errorf("Expected 1 processed message, got %d", count)
		}

		// 检查统计信息
		stats := handler.GetStats()
		if stats.TotalProcessed != 1 {
			t.Errorf("Expected 1 total processed, got %d", stats.TotalProcessed)
		}

		// 停止处理器
		handler.Stop()
	})
}

// TestPeerManager 测试节点管理器
func TestPeerManager(t *testing.T) {
	setupNetworkTestEnvironment()
	defer teardownNetworkTestEnvironment()

	// 创建临时配置文件
	configFile := "test_peer_config.json"
	configContent := `{
		"seed_nodes": ["localhost:3000", "localhost:3001"],
		"max_peers": 10
	}`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	defer os.Remove(configFile)

	manager := peer.NewManager(configFile)
	defer manager.Stop()

	t.Run("PeerOperations", func(t *testing.T) {
		// 获取初始节点
		initialPeers := manager.GetAllPeers()
		if len(initialPeers) != 2 {
			t.Errorf("Expected 2 initial peers, got %d", len(initialPeers))
		}

		// 添加新节点
		newPeer := peer.NewPeer("localhost", 3002)
		manager.AddPeer(newPeer)

		allPeers := manager.GetAllPeers()
		if len(allPeers) != 3 {
			t.Errorf("Expected 3 peers after adding, got %d", len(allPeers))
		}

		// 获取最佳节点
		bestPeers := manager.GetBestPeers(2)
		if len(bestPeers) > 2 {
			t.Errorf("Expected at most 2 best peers, got %d", len(bestPeers))
		}

		// 获取随机节点
		randomPeers := manager.GetRandomPeers(1)
		if len(randomPeers) > 1 {
			t.Errorf("Expected at most 1 random peer, got %d", len(randomPeers))
		}
	})

	t.Run("PeerStats", func(t *testing.T) {
		stats := manager.GetStats()
		if stats["total_peers"] == nil {
			t.Error("Stats should contain total_peers")
		}

		if stats["max_peers"] == nil {
			t.Error("Stats should contain max_peers")
		}
	})
}

// TestNetworkSecurity 测试网络安全模块
func TestNetworkSecurity(t *testing.T) {
	setupNetworkTestEnvironment()
	defer teardownNetworkTestEnvironment()

	nodeAuth, err := security.NewNodeAuth("test-node")
	if err != nil {
		t.Fatalf("Failed to create node auth: %v", err)
	}

	t.Run("NodeAuthentication", func(t *testing.T) {
		// 测试公钥字节获取
		publicKeyBytes, err := nodeAuth.GetPublicKeyBytes()
		if err != nil {
			t.Errorf("Failed to get public key bytes: %v", err)
		}
		if len(publicKeyBytes) == 0 {
			t.Error("Public key bytes should not be empty")
		}

		// 测试签名
		message := []byte("test message")
		signature, err := nodeAuth.SignMessage(message)
		if err != nil {
			t.Errorf("Failed to sign message: %v", err)
		}

		if len(signature) == 0 {
			t.Error("Signature should not be empty")
		}

		// 测试挑战生成
		challenge, err := nodeAuth.GenerateChallenge()
		if err != nil {
			t.Errorf("Failed to generate challenge: %v", err)
		}
		if len(challenge) == 0 {
			t.Error("Challenge should not be empty")
		}
	})

	t.Run("DDoSProtection", func(t *testing.T) {
		blacklistFilter := security.NewBlacklistFilter()
		ddosFilter := security.NewDDoSFilter(10, blacklistFilter)

		// 测试正常请求
		allowed := ddosFilter.ShouldAllow(nil, "192.168.1.1:3000", "test")
		if !allowed {
			t.Error("First request should be allowed")
		}

		// 测试频率限制
		for i := 0; i < 15; i++ {
			ddosFilter.ShouldAllow(nil, "192.168.1.2:3000", "test")
		}

		blocked := ddosFilter.ShouldAllow(nil, "192.168.1.2:3000", "test")
		if blocked {
			t.Error("Request should be blocked after rate limit")
		}
	})
}

// TestIntegrationNetwork 集成测试
func TestIntegrationNetwork(t *testing.T) {
	setupNetworkTestEnvironment()
	defer teardownNetworkTestEnvironment()

	t.Run("BasicNetworkFlow", func(t *testing.T) {
		// 这个测试模拟基本的网络流程
		// 由于需要真实的网络连接，这里只做基础的组件集成测试

		// 创建消息处理器
		msgHandler := message.NewHandler(1)
		err := msgHandler.Start()
		if err != nil {
			t.Errorf("Failed to start message handler: %v", err)
		}
		defer msgHandler.Stop()

		// 创建连接管理器
		connManager := connection.NewManager(nil)
		err = connManager.Start()
		if err != nil {
			t.Errorf("Failed to start connection manager: %v", err)
		}
		defer connManager.Stop()

		// 验证组件正常运行
		if !msgHandler.IsRunning() {
			t.Error("Message handler should be running")
		}

		managerStats := connManager.GetManagerStats()
		if !managerStats["is_running"].(bool) {
			t.Error("Connection manager should be running")
		}
	})
}

// TestNetworkWithBlockchain 测试网络与区块链的集成
func TestNetworkWithBlockchain(t *testing.T) {
	setupNetworkTestEnvironment()
	defer teardownNetworkTestEnvironment()

	t.Run("BlockchainNetworkIntegration", func(t *testing.T) {
		// 创建测试区块链
		bc := blockchain.NewBlockchain("test-address", testNodeID)
		defer bc.DB.Close()

		// 验证区块链创建
		bestHeight := bc.GetBestHeight()
		if bestHeight < 0 {
			t.Error("Blockchain should have at least genesis block")
		}

		// 测试网络消息类型
		testTypes := []string{"version", "block", "tx", "inv", "getblocks", "getdata", "addr"}

		for _, msgType := range testTypes {
			// 验证消息类型转换
			cmdBytes := CommandToBytes(msgType)
			convertedType := BytesToCommand(cmdBytes)

			if convertedType != msgType {
				t.Errorf("Message type conversion failed: %s != %s", msgType, convertedType)
			}
		}
	})
}

// MockServer 用于测试的模拟服务器
type MockServer struct {
	address  string
	listener net.Listener
	running  bool
	mutex    sync.Mutex
}

func NewMockServer(address string) *MockServer {
	return &MockServer{
		address: address,
		running: false,
	}
}

func (s *MockServer) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.running {
		return fmt.Errorf("server already running")
	}

	ln, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	s.listener = ln
	s.running = true

	go s.handleConnections()
	return nil
}

func (s *MockServer) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	return s.listener.Close()
}

func (s *MockServer) handleConnections() {
	for s.running {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.running {
				continue
			}
			break
		}

		go func(c net.Conn) {
			defer c.Close()
			// 简单的echo服务器，用于测试连接
			buffer := make([]byte, 1024)
			n, err := c.Read(buffer)
			if err == nil && n > 0 {
				c.Write(buffer[:n])
			}
		}(conn)
	}
}

// TestMockNetworkServer 测试模拟网络服务器
func TestMockNetworkServer(t *testing.T) {
	setupNetworkTestEnvironment()
	defer teardownNetworkTestEnvironment()

	t.Run("MockServerOperations", func(t *testing.T) {
		server := NewMockServer("localhost:0") // 使用随机端口

		err := server.Start()
		if err != nil {
			t.Errorf("Failed to start mock server: %v", err)
		}

		// 获取实际监听地址
		actualAddr := server.listener.Addr().String()

		// 测试连接
		conn, err := net.DialTimeout("tcp", actualAddr, 2*time.Second)
		if err != nil {
			t.Errorf("Failed to connect to mock server: %v", err)
		} else {
			conn.Close()
		}

		err = server.Stop()
		if err != nil {
			t.Errorf("Failed to stop mock server: %v", err)
		}
	})
}
