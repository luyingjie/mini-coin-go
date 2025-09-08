package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync"
	"time"
)

// NodeAuth 节点身份验证
type NodeAuth struct {
	nodeID     string
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	peers      map[string]*rsa.PublicKey // 已认证节点的公钥
	mutex      sync.RWMutex
}

// AuthMessage 认证消息结构
type AuthMessage struct {
	NodeID    string    `json:"node_id"`
	Timestamp int64     `json:"timestamp"`
	Challenge []byte    `json:"challenge"`
	Signature []byte    `json:"signature"`
	PublicKey []byte    `json:"public_key"`
}

// NewNodeAuth 创建节点认证实例
func NewNodeAuth(nodeID string) (*NodeAuth, error) {
	// 生成RSA密钥对
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("生成RSA密钥失败: %v", err)
	}
	
	return &NodeAuth{
		nodeID:     nodeID,
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		peers:      make(map[string]*rsa.PublicKey),
	}, nil
}

// GetPublicKeyBytes 获取公钥字节数组
func (na *NodeAuth) GetPublicKeyBytes() ([]byte, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(na.publicKey)
	if err != nil {
		return nil, fmt.Errorf("序列化公钥失败: %v", err)
	}
	
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	
	return publicKeyPEM, nil
}

// LoadPublicKeyFromBytes 从字节数组加载公钥
func (na *NodeAuth) LoadPublicKeyFromBytes(keyBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, fmt.Errorf("解码PEM失败")
	}
	
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析公钥失败: %v", err)
	}
	
	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("不是RSA公钥")
	}
	
	return rsaPublicKey, nil
}

// CreateAuthMessage 创建认证消息
func (na *NodeAuth) CreateAuthMessage(challenge []byte) (*AuthMessage, error) {
	// 创建消息内容
	message := fmt.Sprintf("%s:%d:%x", na.nodeID, time.Now().Unix(), challenge)
	messageHash := sha256.Sum256([]byte(message))
	
	// 使用私钥签名
	signature, err := rsa.SignPKCS1v15(rand.Reader, na.privateKey, 
		0, messageHash[:])
	if err != nil {
		return nil, fmt.Errorf("签名失败: %v", err)
	}
	
	// 获取公钥字节
	publicKeyBytes, err := na.GetPublicKeyBytes()
	if err != nil {
		return nil, fmt.Errorf("获取公钥失败: %v", err)
	}
	
	return &AuthMessage{
		NodeID:    na.nodeID,
		Timestamp: time.Now().Unix(),
		Challenge: challenge,
		Signature: signature,
		PublicKey: publicKeyBytes,
	}, nil
}

// VerifyAuthMessage 验证认证消息
func (na *NodeAuth) VerifyAuthMessage(authMsg *AuthMessage) error {
	// 检查时间戳（5分钟内有效）
	if time.Now().Unix()-authMsg.Timestamp > 300 {
		return fmt.Errorf("认证消息已过期")
	}
	
	// 加载公钥
	publicKey, err := na.LoadPublicKeyFromBytes(authMsg.PublicKey)
	if err != nil {
		return fmt.Errorf("加载公钥失败: %v", err)
	}
	
	// 重建消息内容
	message := fmt.Sprintf("%s:%d:%x", authMsg.NodeID, authMsg.Timestamp, authMsg.Challenge)
	messageHash := sha256.Sum256([]byte(message))
	
	// 验证签名
	err = rsa.VerifyPKCS1v15(publicKey, 0, messageHash[:], authMsg.Signature)
	if err != nil {
		return fmt.Errorf("签名验证失败: %v", err)
	}
	
	// 保存已验证的公钥
	na.mutex.Lock()
	na.peers[authMsg.NodeID] = publicKey
	na.mutex.Unlock()
	
	return nil
}

// GenerateChallenge 生成挑战
func (na *NodeAuth) GenerateChallenge() ([]byte, error) {
	challenge := make([]byte, 32)
	_, err := rand.Read(challenge)
	if err != nil {
		return nil, fmt.Errorf("生成挑战失败: %v", err)
	}
	return challenge, nil
}

// AuthenticatePeer 认证节点
func (na *NodeAuth) AuthenticatePeer(peerID string, challenge []byte) (*AuthMessage, error) {
	// 创建认证消息
	authMsg, err := na.CreateAuthMessage(challenge)
	if err != nil {
		return nil, fmt.Errorf("创建认证消息失败: %v", err)
	}
	
	return authMsg, nil
}

// IsAuthenticated 检查节点是否已认证
func (na *NodeAuth) IsAuthenticated(peerID string) bool {
	na.mutex.RLock()
	defer na.mutex.RUnlock()
	
	_, exists := na.peers[peerID]
	return exists
}

// GetAuthenticatedPeers 获取已认证的节点列表
func (na *NodeAuth) GetAuthenticatedPeers() []string {
	na.mutex.RLock()
	defer na.mutex.RUnlock()
	
	var peers []string
	for peerID := range na.peers {
		peers = append(peers, peerID)
	}
	return peers
}

// RemovePeer 移除节点认证
func (na *NodeAuth) RemovePeer(peerID string) {
	na.mutex.Lock()
	defer na.mutex.Unlock()
	
	delete(na.peers, peerID)
}

// SignMessage 对消息进行签名
func (na *NodeAuth) SignMessage(message []byte) ([]byte, error) {
	messageHash := sha256.Sum256(message)
	
	signature, err := rsa.SignPKCS1v15(rand.Reader, na.privateKey, 
		0, messageHash[:])
	if err != nil {
		return nil, fmt.Errorf("签名失败: %v", err)
	}
	
	return signature, nil
}

// VerifyMessageSignature 验证消息签名
func (na *NodeAuth) VerifyMessageSignature(peerID string, message []byte, signature []byte) error {
	na.mutex.RLock()
	publicKey, exists := na.peers[peerID]
	na.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("节点未认证: %s", peerID)
	}
	
	messageHash := sha256.Sum256(message)
	
	err := rsa.VerifyPKCS1v15(publicKey, 0, messageHash[:], signature)
	if err != nil {
		return fmt.Errorf("签名验证失败: %v", err)
	}
	
	return nil
}

// GetNodeID 获取节点ID
func (na *NodeAuth) GetNodeID() string {
	return na.nodeID
}

// GetAuthStats 获取认证统计信息
func (na *NodeAuth) GetAuthStats() map[string]interface{} {
	na.mutex.RLock()
	defer na.mutex.RUnlock()
	
	return map[string]interface{}{
		"node_id":            na.nodeID,
		"authenticated_peers": len(na.peers),
		"peer_list":          na.GetAuthenticatedPeers(),
	}
}