package blockchain

import (
	"time"
)

// Block 是区块链的基本组成单位
type Block struct {
	Timestamp     int64  // 时间戳, 区块创建的时间
	Data          []byte // 区块存储的实际有效信息，例如交易
	PrevBlockHash []byte // 前一个区块的哈希值
	Hash          []byte // 当前区块的哈希值
	Nonce         int    // 工作量证明的计数器
}

// NewBlock 创建并返回一个新区块
func NewBlock(data string, prevBlockHash []byte) *Block {
	block := &Block{
		Timestamp:     time.Now().Unix(),
		Data:          []byte(data),
		PrevBlockHash: prevBlockHash,
		Hash:          []byte{},
		Nonce:         0,
	}
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run() // 通过挖矿得到 nonce 和 hash

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

// NewGenesisBlock 创建并返回创世区块
func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}
