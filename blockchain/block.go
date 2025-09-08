package blockchain

import (
	"bytes"
	"encoding/gob"
	"log"
	"time"
)

// Block 是区块链的基本组成单位
type Block struct {
	Timestamp     int64          // 时间戳, 区块创建的时间
	Transactions  []*Transaction // 区块存储的实际有效信息，例如交易
	PrevBlockHash []byte         // 前一个区块的哈希值
	Hash          []byte         // 当前区块的哈希值
	Nonce         int            // 工作量证明的计数器
	Height        int            // 区块高度
}

// Serialize 将区块序列化为一个字节切片
func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}

// DeserializeBlock 将字节切片反序列化为一个区块
func DeserializeBlock(d []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}

	return &block
}

func (b *Block) HashTransactions() []byte {
	var transactions [][]byte

	for _, tx := range b.Transactions {
		transactions = append(transactions, tx.ID)
	}

	mTree := NewMerkleTree(transactions)

	return mTree.RootNode.Data
}

// NewBlock 创建并返回一个新区块
func NewBlock(transactions []*Transaction, prevBlockHash []byte, height int) *Block {
	block := &Block{
		Timestamp:     time.Now().Unix(),
		Transactions:  transactions,
		PrevBlockHash: prevBlockHash,
		Hash:          []byte{},
		Nonce:         0,
		Height:        height,
	}
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run() // 通过挖矿得到 nonce 和 hash

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

// NewGenesisBlock 创建并返回创世区块
func NewGenesisBlock(coinbaseTx *Transaction) *Block {
	return NewBlock([]*Transaction{coinbaseTx}, []byte{}, 0)
}
