package blockchain

// Blockchain 是一个指向 Block 的指针数组
type Blockchain struct {
	Blocks []*Block
}

// AddBlock 向链上添加一个新区块
func (bc *Blockchain) AddBlock(data string) {
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock(data, prevBlock.Hash)
	bc.Blocks = append(bc.Blocks, newBlock)
}

// NewBlockchain 创建一个包含创世区块的区块链
func NewBlockchain() *Blockchain {
	return &Blockchain{[]*Block{NewGenesisBlock()}}
}
