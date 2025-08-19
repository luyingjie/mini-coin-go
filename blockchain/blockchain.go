package blockchain

import (
	"log"

	"go.etcd.io/bbolt"
)

const dbFile = "blockchain.db"
const blocksBucket = "blocks"

// Blockchain 结构体现在只包含数据库连接和链的末端哈希
type Blockchain struct {
	tip []byte
	DB  *bbolt.DB
}

// AddBlock 将新区块保存到数据库中
func (bc *Blockchain) AddBlock(data string) {
	var lastHash []byte

	// 查看数据库以获取最后一个区块的哈希
	err := bc.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	newBlock := NewBlock(data, lastHash)

	// 将新区块存入数据库并更新 "l" 键
	err = bc.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		err := b.Put(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			log.Panic(err)
		}
		err = b.Put([]byte("l"), newBlock.Hash)
		if err != nil {
			log.Panic(err)
		}
		bc.tip = newBlock.Hash
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

// NewBlockchain 创建一个新的区块链数据库，如果不存在的话
func NewBlockchain() *Blockchain {
	var tip []byte
	db, err := bbolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		if b == nil {
			// 如果 bucket 不存在，说明链是新的
			genesis := NewGenesisBlock()
			b, err := tx.CreateBucket([]byte(blocksBucket))
			if err != nil {
				log.Panic(err)
			}
			err = b.Put(genesis.Hash, genesis.Serialize())
			if err != nil {
				log.Panic(err)
			}
			err = b.Put([]byte("l"), genesis.Hash)
			if err != nil {
				log.Panic(err)
			}
			tip = genesis.Hash
		} else {
			// 如果 bucket 已存在，读取 "l" 键
			tip = b.Get([]byte("l"))
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	bc := Blockchain{tip, db}

	return &bc
}

// BlockchainIterator 用于遍历区块链
type BlockchainIterator struct {
	currentHash []byte
	DB          *bbolt.DB
}

// Iterator 创建并返回一个区块链迭代器
func (bc *Blockchain) Iterator() *BlockchainIterator {
	return &BlockchainIterator{bc.tip, bc.DB}
}

// Next 返回链中的下一个区块（从后往前）
func (i *BlockchainIterator) Next() *Block {
	var block *Block

	err := i.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(i.currentHash)
		block = DeserializeBlock(encodedBlock)
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	i.currentHash = block.PrevBlockHash

	return block
}
