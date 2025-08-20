package blockchain

import (
	"crypto/sha256"
)

// MerkleTree 表示一个默克尔树
type MerkleTree struct {
	RootNode *MerkleNode
}

// MerkleNode 表示一个默克尔树节点
type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

// NewMerkleTree 从数据序列中创建一个新的默克尔树
func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []MerkleNode

	// If number of transactions is odd, duplicate the last one to make it even
	// 如果交易数量是奇数，则复制最后一个以使其变为偶数
	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	for _, datum := range data {
		node := NewMerkleNode(nil, nil, datum)
		nodes = append(nodes, *node)
	}

	for len(nodes) > 1 {
		var newLevel []MerkleNode

		for i := 0; i < len(nodes); i += 2 {
			node := NewMerkleNode(&nodes[i], &nodes[i+1], nil)
			newLevel = append(newLevel, *node)
		}

		nodes = newLevel
	}

	mTree := MerkleTree{&nodes[0]}

	return &mTree
}

// NewMerkleNode 创建一个新的默克尔树节点
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	mNode := MerkleNode{}

	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		mNode.Data = hash[:]
	} else {
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		mNode.Data = hash[:]
	}

	mNode.Left = left
	mNode.Right = right

	return &mNode
}
