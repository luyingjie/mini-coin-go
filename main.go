package main

import (
	"mini-coin-go/blockchain"
)

func main() {
	bc := blockchain.NewBlockchain()
	defer bc.DB.Close() // 确保程序结束时关闭数据库

	cli := CLI{bc}
	cli.Run()
}
