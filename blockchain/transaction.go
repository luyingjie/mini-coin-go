package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"log"
)

// TXInput 结构:
type TXInput struct {
	Txid      []byte // 引用来源交易的 ID (哈希)
	Vout      int    // 引用来源交易的某个输出的索引
	ScriptSig string // 解锁脚本，这里我们可以简化为发送方的地址
}

// TXOutput 结构:
type TXOutput struct {
	Value        int    // 金额
	ScriptPubKey []byte // 锁定脚本，这里我们可以简化为接收方的地址
}

// IsLockedWithKey 检查输出是否可以用提供的密钥解锁
func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	// return bytes.Compare(out.ScriptPubKey, pubKeyHash) == 0
	return bytes.Equal(out.ScriptPubKey, pubKeyHash)
}

// TXOutputs 收集 TXOutput
type TXOutputs struct {
	Outputs []TXOutput
}

// Serialize 序列化 TXOutputs
func (outs TXOutputs) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)

	err := encoder.Encode(outs)
	if err != nil {
		log.Panic(err)
	}

	return res.Bytes()
}

// DeserializeOutputs 反序列化 TXOutputs
func DeserializeOutputs(data []byte) TXOutputs {
	var outputs TXOutputs

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&outputs)
	if err != nil {
		log.Panic(err)
	}

	return outputs
}

// Transaction 结构:
type Transaction struct {
	ID   []byte     // 交易的唯一标识 (哈希)
	Vin  []TXInput  // 交易输入
	Vout []TXOutput // 交易输出
}

// Hash 计算交易的哈希值
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)
	err := encoder.Encode(txCopy)
	if err != nil {
		log.Panic(err)
	}

	hash = sha256.Sum256(res.Bytes())

	return hash[:]
}

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

// NewCoinbaseTX 创建并返回一个 Coinbase 交易
func NewCoinbaseTX(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Reward to %s", to)
	}

	// Coinbase 交易没有输入，Txid 为空，Vout 为 -1
	in := TXInput{[]byte{}, -1, data}
	out := TXOutput{100, []byte(to)} // 奖励 100 个币

	tx := Transaction{nil, []TXInput{in}, []TXOutput{out}}
	tx.ID = tx.Hash()

	return &tx
}
