package main

import (
	"fmt"
	"log"
	"os"

	"mini-coin-go/blockchain"
	"mini-coin-go/wallet"

	"go.etcd.io/bbolt"
)

func debugTransaction() {
	// 清理测试文件
	os.Remove("blockchain.db")
	os.Remove("wallet.dat")

	fmt.Println("=== 创建钱包和区块链 ===")

	// 创建钱包
	wallets, err := wallet.NewWallets()
	if err != nil {
		log.Panic(err)
	}

	addressA := wallets.CreateWallet()
	addressB := wallets.CreateWallet()
	wallets.SaveToFile()

	fmt.Printf("钱包A地址: %s\n", addressA)
	fmt.Printf("钱包B地址: %s\n", addressB)

	// 创建区块链，给A地址100币
	bc := blockchain.NewBlockchain(addressA)
	defer bc.DB.Close()

	utxoSet := blockchain.UTXOSet{bc}
	utxoSet.Reindex()

	// 检查初始余额
	fmt.Println("\n=== 初始余额 ===")
	balanceA := getBalance(addressA, &utxoSet)
	balanceB := getBalance(addressB, &utxoSet)
	fmt.Printf("钱包A余额: %d\n", balanceA)
	fmt.Printf("钱包B余额: %d\n", balanceB)

	// 发送10币从A到B
	fmt.Println("\n=== 发送交易：A -> B (10币) ===")

	// 检查A的可花费输出
	pubKeyHashA := blockchain.Base58Decode([]byte(addressA))
	pubKeyHashA = pubKeyHashA[1 : len(pubKeyHashA)-4]
	acc, validOutputs := utxoSet.FindSpendableOutputs(pubKeyHashA, 10)
	fmt.Printf("找到的可花费输出: 总金额=%d\n", acc)
	for txid, outs := range validOutputs {
		fmt.Printf("  TxID: %s (长度:%d), Outputs: %v\n", txid, len(txid), outs)
	}

	tx := blockchain.NewUTXOTransaction(addressA, addressB, 10, &utxoSet)
	fmt.Printf("交易ID: %x\n", tx.ID)

	// 打印交易详情
	fmt.Println("交易输入:")
	for i, in := range tx.Vin {
		fmt.Printf("  输入%d: TxID=%x (长度:%d), Vout=%d, ScriptSig=%s\n", i, in.Txid, len(in.Txid), in.Vout, in.ScriptSig)
	}

	fmt.Println("交易输出:")
	for i, out := range tx.Vout {
		fmt.Printf("  输出%d: Value=%d, ScriptPubKey=%x\n", i, out.Value, out.ScriptPubKey)
	}

	// 挖矿（不给奖励）
	bc.MineBlockWithoutReward([]*blockchain.Transaction{tx})
	utxoSet.Reindex()

	// 检查交易后余额
	fmt.Println("\n=== 交易后余额 ===")
	balanceA = getBalance(addressA, &utxoSet)
	balanceB = getBalance(addressB, &utxoSet)
	fmt.Printf("钱包A余额: %d (期望: 90)\n", balanceA)
	fmt.Printf("钱包B余额: %d (期望: 10)\n", balanceB)

	// 检查原始UTXO数据
	fmt.Println("\n=== 原始UTXO数据库内容 ===")
	debugRawUTXO(bc)

	// 检查FindUTXO的结果
	fmt.Println("\n=== FindUTXO结果 ===")
	utxos := bc.FindUTXO()
	for txIDStr, outs := range utxos {
		fmt.Printf("TxID (string): %s (长度: %d)\n", txIDStr, len(txIDStr))
		fmt.Printf("TxID (hex): %x\n", []byte(txIDStr))
		for i, out := range outs.Outputs {
			fmt.Printf("  输出%d: Value=%d, ScriptPubKey=%x\n", i, out.Value, out.ScriptPubKey)
		}
	}

	// 检查UTXO详情
	fmt.Println("\n=== UTXO详情 ===")
	debugUTXO(addressA, &utxoSet)
	debugUTXO(addressB, &utxoSet)
}

func getBalance(address string, utxoSet *blockchain.UTXOSet) int {
	pubKeyHash := blockchain.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	utxos := utxoSet.FindUTXO(pubKeyHash)

	balance := 0
	for _, out := range utxos {
		balance += out.Value
	}
	return balance
}

func debugUTXO(address string, utxoSet *blockchain.UTXOSet) {
	pubKeyHash := blockchain.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	utxos := utxoSet.FindUTXO(pubKeyHash)

	fmt.Printf("地址 %s 的UTXO:\n", address[len(address)-6:]) // 只显示最后6个字符
	for i, out := range utxos {
		fmt.Printf("  UTXO%d: Value=%d, ScriptPubKey=%x\n", i+1, out.Value, out.ScriptPubKey)
	}
	fmt.Printf("  总计: %d 个UTXO\n", len(utxos))
}

func main() {
	debugTransaction()
}

func debugRawUTXO(bc *blockchain.Blockchain) {
	err := bc.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("chainstate"))
		if b == nil {
			fmt.Println("没有找到 chainstate bucket")
			return nil
		}

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Printf("Raw Key (bytes): %x (长度: %d)\n", k, len(k))
			fmt.Printf("Raw Key (string): %s\n", string(k))

			outs := blockchain.DeserializeOutputs(v)
			for i, out := range outs.Outputs {
				fmt.Printf("  输出%d: Value=%d, ScriptPubKey=%x\n", i, out.Value, out.ScriptPubKey)
			}
			fmt.Println()
		}
		return nil
	})

	if err != nil {
		log.Panic(err)
	}
}
