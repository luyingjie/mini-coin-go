package cmd

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"mini-coin-go/blockchain"
	"mini-coin-go/wallet"
)

// CLI handles command line arguments 命令行接口处理命令行参数
type CLI struct {
	// bc *blockchain.Blockchain // 移除此字段
}

// printUsage 打印用法说明
func (cli *CLI) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  createblockchain -address ADDRESS - Create a blockchain and send genesis reward to ADDRESS")
	fmt.Println("  createwallet - Generates a new key-pair and saves it into the wallet file")
	fmt.Println("  getbalance -address ADDRESS - Get balance of ADDRESS")
	fmt.Println("  listaddresses - Lists all addresses from the wallet file")
	fmt.Println("  printchain - Print all the blocks of the blockchain")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT - Send AMOUNT of coins from FROM address to TO")
}

// validateArgs 确保命令行参数有效
func (cli *CLI) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

// createBlockchain 创建区块链
func (cli *CLI) createBlockchain(address string) {
	if !blockchain.ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bc := blockchain.NewBlockchain(address)
	defer bc.DB.Close()

	utxoSet := blockchain.UTXOSet{bc}
	utxoSet.Reindex()

	fmt.Println("Done!")
}

// createWallet 创建钱包
func (cli *CLI) createWallet() {
	wallets, _ := wallet.NewWallets()
	address := wallets.CreateWallet()
	wallets.SaveToFile()

	fmt.Printf("Your new address: %s\n", address)
}

// getBalance 查询余额
func (cli *CLI) getBalance(address string) {
	if !blockchain.ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bc := blockchain.NewBlockchain("") // Pass empty string as address for existing blockchain
	defer bc.DB.Close()

	utxoSet := blockchain.UTXOSet{bc}
	utxoSet.Reindex() // Reindex UTXO set for current blockchain state
	balance := 0
	pubKeyHash := blockchain.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	utxos := utxoSet.FindUTXO(pubKeyHash)

	for _, out := range utxos {
		balance += out.Value
	}

	fmt.Printf("Balance of '%s': %d\n", address, balance)
}

// listAddresses 列出所有地址
func (cli *CLI) listAddresses() {
	wallets, err := wallet.NewWallets()
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

// printChain 打印区块链
func (cli *CLI) printChain() {
	bc := blockchain.NewBlockchain("") // Pass empty string as address for existing blockchain
	defer bc.DB.Close()

	bci := bc.Iterator()

	for {
		block := bci.Next()

		fmt.Printf("============ Block %x ============%s", block.Hash, "\n")
		fmt.Printf("Prev. hash: %x%s", block.PrevBlockHash, "\n")
		pow := blockchain.NewProofOfWork(block)
		fmt.Printf("PoW: %s%s", strconv.FormatBool(pow.Validate()), "\n")
		fmt.Printf("Timestamp: %d%s", block.Timestamp, "\n")
		fmt.Printf("Nonce: %d%s", block.Nonce, "\n")
		fmt.Println("Transactions:")
		for _, tx := range block.Transactions {
			fmt.Printf("%x%s", tx.ID, "\n")
			fmt.Printf("  -Vins:%s", "\n")
			for _, in := range tx.Vin {
				fmt.Printf("    TxID: %x, Vout: %d, ScriptSig: %s%s", in.Txid, in.Vout, in.ScriptSig, "\n")
			}
			fmt.Printf("  -Vouts:%s", "\n")
			for _, out := range tx.Vout {
				fmt.Printf("    Value: %d, ScriptPubKey: %x%s", out.Value, out.ScriptPubKey, "\n")
			}
		}
		fmt.Println()

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}

// send 发送交易
func (cli *CLI) send(from, to string, amount int) {
	if !blockchain.ValidateAddress(from) {
		log.Panic("ERROR: Sender address is not valid")
	}
	if !blockchain.ValidateAddress(to) {
		log.Panic("ERROR: Recipient address is not valid")
	}

	bc := blockchain.NewBlockchain("") // Pass empty string as address for existing blockchain
	defer bc.DB.Close()

	utxoSet := blockchain.UTXOSet{bc}
	utxoSet.Reindex() // Reindex UTXO set for current blockchain state
	tx := blockchain.NewUTXOTransaction(from, to, amount, &utxoSet)

	// cbTx := blockchain.NewCoinbaseTX(from, "") // Remove coinbase transaction from send
	txns := []*blockchain.Transaction{tx}

	bc.MineBlock(txns, from) // Pass 'from' as miner address
	// utxoSet.Update(bc.Iterator().Next()) // This update is handled by Reindex now

	fmt.Println("Success!")
}

// Run 解析命令行参数并执行相应的命令
func (cli *CLI) Run() {
	cli.validateArgs()

	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)

	createBlockchainAddress := createBlockchainCmd.String("address", "", "接收创世区块奖励的地址")
	getBalanceAddress := getBalanceCmd.String("address", "", "查询余额的地址")
	sendFrom := sendCmd.String("from", "", "源钱包地址")
	sendTo := sendCmd.String("to", "", "目标钱包地址")
	sendAmount := sendCmd.Int("amount", 0, "发送金额")

	switch os.Args[1] {
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			os.Exit(1)
		}
		cli.createBlockchain(*createBlockchainAddress)
	}

	if createWalletCmd.Parsed() {
		cli.createWallet()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddress)
	}

	if listAddressesCmd.Parsed() {
		cli.listAddresses()
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			os.Exit(1)
		}
		cli.send(*sendFrom, *sendTo, *sendAmount)
	}
}
