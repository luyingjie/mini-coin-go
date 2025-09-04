package blockchain

import (
	"encoding/hex"
	"log"

	"go.etcd.io/bbolt"
)

const dbFile = "blockchain.db"
const blocksBucket = "blocks"
const utxoBucket = "chainstate"

// UTXOSet 表示 UTXO 集合
type UTXOSet struct {
	Blockchain *Blockchain
}

// FindSpendableOutputs 查找并返回未花费的输出，以便在输入中引用
func (u UTXOSet) FindSpendableOutputs(pubkeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	accumulated := 0

	u.Blockchain.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

	Work:
		for k, v := c.First(); k != nil; k, v = c.Next() {
			txID := string(k) // k已经是十六进制字符串，直接使用
			outs := DeserializeOutputs(v)

			for outIdx, out := range outs.Outputs {
				if out.IsLockedWithKey(pubkeyHash) {
					accumulated += out.Value
					unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)

					if accumulated >= amount {
						break Work
					}
				}
			}
		}

		return nil
	})

	return accumulated, unspentOutputs
}

// FindUTXO 查找所有未花费的交易输出并返回已移除花费输出的交易
func (u UTXOSet) FindUTXO(pubkeyHash []byte) []TXOutput {
	var UTXOs []TXOutput

	u.Blockchain.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			outs := DeserializeOutputs(v)

			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubkeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}

		return nil
	})

	return UTXOs
}

// Reindex 重建 UTXO 集合
func (u UTXOSet) Reindex() {
	err := u.Blockchain.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		if b != nil {
			err := tx.DeleteBucket([]byte(utxoBucket))
			if err != nil {
				log.Panic(err)
			}
		}

		_, err := tx.CreateBucket([]byte(utxoBucket))
		if err != nil {
			log.Panic(err)
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	utxos := u.Blockchain.FindUTXO()

	err = u.Blockchain.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		for txID, outs := range utxos {
			err := b.Put([]byte(txID), outs.Serialize()) // txID已经是十六进制字符串，直接使用
			if err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
}

// Update 使用区块中的交易更新 UTXO 集合
// 该区块是区块链的最后一个区块
func (u UTXOSet) Update(block *Block) {
	err := u.Blockchain.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))

		for _, tx := range block.Transactions {
			if tx.IsCoinbase() == false {
				for _, vin := range tx.Vin {
					updatedOuts := TXOutputs{[]TXOutput{}}
					outsBytes := b.Get(vin.Txid)
					outs := DeserializeOutputs(outsBytes)

					for outIdx, out := range outs.Outputs {
						if outIdx != vin.Vout {
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}

					if len(updatedOuts.Outputs) == 0 {
						err := b.Delete(vin.Txid)
						if err != nil {
							log.Panic(err)
						}
					} else {
						err := b.Put(vin.Txid, updatedOuts.Serialize())
						if err != nil {
							log.Panic(err)
						}
					}
				}
			}
			err := b.Put(tx.ID, TXOutputs{tx.Vout}.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

// Blockchain 结构体现在只包含数据库连接和链的末端哈希
type Blockchain struct {
	tip []byte
	DB  *bbolt.DB
}

// AddBlock 将新区块保存到数据库中
func (bc *Blockchain) MineBlock(transactions []*Transaction, minerAddress string) {
	var lastHash []byte

	// 查看数据���以获取最后一个区块的哈希
	err := bc.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	// Create a coinbase transaction for the miner
	cbtx := NewCoinbaseTX(minerAddress, "")
	transactions = append([]*Transaction{cbtx}, transactions...)

	newBlock := NewBlock(transactions, lastHash)

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

// MineBlockWithoutReward 不给矿工奖励的情况下挖矿新区块（用于纯转账交易）
func (bc *Blockchain) MineBlockWithoutReward(transactions []*Transaction) {
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

	// 不创建 Coinbase 交易，直接使用传入的交易
	newBlock := NewBlock(transactions, lastHash)

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

// 这个方法将会替代 MineBlock 和 MineBlockWithoutReward 方法。
// MineBlockUnified 将新区块保存到数据库中
// 如果 minerAddress 不为空, 则会创建一笔 Coinbase 交易作为矿工奖励
func (bc *Blockchain) MineBlockUnified(transactions []*Transaction, minerAddress string) {
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

	// 如果提供了矿工地址，则创建并添加 Coinbase 交易
	if minerAddress != "" {
		cbtx := NewCoinbaseTX(minerAddress, "")
		transactions = append([]*Transaction{cbtx}, transactions...)
	}

	newBlock := NewBlock(transactions, lastHash)

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
func NewBlockchain(address string) *Blockchain {
	var tip []byte
	db, err := bbolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		if b == nil {
			// 如果 bucket 不存在，说明链是新的
			cbtx := NewCoinbaseTX(address, "") // 创世区块的 Coinbase 交易
			genesis := NewBlock([]*Transaction{cbtx}, []byte{})
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
	// utxoSet := UTXOSet{&bc}
	// utxoSet.Reindex()

	return &bc
}

// FindUTXO 检索未使用的交易输出
func (bc *Blockchain) FindUTXO() map[string]TXOutputs {
	utxo := make(map[string]TXOutputs)
	spentTXOs := make(map[string][]int)

	bci := bc.Iterator()

	// 第一遍：收集所有已花费的输出
	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			if tx.IsCoinbase() == false {
				for _, in := range tx.Vin {
					inTxID := hex.EncodeToString(in.Txid)
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	// 重新创建迭代器进行第二遍
	bci = bc.Iterator()

	// 第二遍：收集未花费的输出
	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			outs := TXOutputs{}
			txIDHex := hex.EncodeToString(tx.ID)
			for outIdx, out := range tx.Vout {
				// 检查这个输出是否被花费了
				isSpent := false
				if spentOuts, ok := spentTXOs[txIDHex]; ok {
					for _, spentOutIdx := range spentOuts {
						if spentOutIdx == outIdx {
							isSpent = true
							break
						}
					}
				}

				// 如果没有被花费，就是 UTXO
				if !isSpent {
					outs.Outputs = append(outs.Outputs, out)
				}
			}
			utxo[txIDHex] = outs // 使用十六进制格式作为key，与Reindex保持一致
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return utxo
}

// NewUTXOTransaction 创建一个新的交易
func NewUTXOTransaction(from, to string, amount int, UTXOSet *UTXOSet) *Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	pubKeyHash := Base58Decode([]byte(from))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]

	acc, validOutputs := UTXOSet.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		log.Panic("ERROR: Not enough funds")
	}

	// Build a list of inputs
	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		if err != nil {
			log.Panic(err)
		}

		for _, out := range outs {
			input := TXInput{txID, out, from}
			inputs = append(inputs, input)
		}
	}

	// Build a list of outputs
	outputs = append(outputs, TXOutput{amount, Base58Decode([]byte(to))[1 : len(Base58Decode([]byte(to)))-4]})
	if acc > amount {
		outputs = append(outputs, TXOutput{acc - amount, Base58Decode([]byte(from))[1 : len(Base58Decode([]byte(from)))-4]}) // Change
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()

	return &tx
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
