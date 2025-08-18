package main

import (
	"fmt"
	"mini-coin-go/blockchain" // 导入我们自己的 blockchain 包
)

func main() {
	bc := blockchain.NewBlockchain()

	fmt.Println("Adding first block...")
	bc.AddBlock("Send 1 BTC to Ivan")

	fmt.Println("Adding second block...")
	bc.AddBlock("Send 2 more BTC to Ivan")

	fmt.Println("Blockchain details:")
	for _, block := range bc.Blocks {
		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)

		// 验证每个区块的工作量证明
		pow := blockchain.NewProofOfWork(block)
		fmt.Printf("PoW: %t\n", pow.Validate())
		fmt.Println()
	}
}