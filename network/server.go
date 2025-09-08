package network

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"

	"mini-coin-go/blockchain"
)

const (
	protocol      = "tcp"
	commandLength = 12
)

var (
	// nodeAddress 当前节点地址
	nodeAddress string
	// miningAddress 挖矿地址
	miningAddress string
	// KnownNodes 已知节点
	KnownNodes = []string{"localhost:3000"}
	// blocksInTransit 用于存储正在传输的区块
	blocksInTransit = [][]byte{}
	// mempool 内存池
	mempool = make(map[string]blockchain.Transaction)
)

// StartServer 启动服务器
func StartServer(nodeID, minerAddress string) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	miningAddress = minerAddress
	ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		log.Panic(err)
	}
	defer ln.Close()

	bc := blockchain.NewBlockchain(minerAddress, nodeID)

	// 如果当前节点不是中心节点，则向中心节点发送版本信息
	if nodeAddress != KnownNodes[0] {
		sendVersion(KnownNodes[0], bc)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}
		go handleConnection(conn, bc)
	}
}

// handleConnection 处理连接
func handleConnection(conn net.Conn, bc *blockchain.Blockchain) {
	request, err := io.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}
	command := BytesToCommand(request[:commandLength])
	fmt.Printf("Received %s command\n", command)

	switch command {
	case "addr":
		handleAddr(request)
	case "block":
		handleBlock(request, bc)
	case "inv":
		handleInv(request, bc)
	case "getblocks":
		handleGetBlocks(request, bc)
	case "getdata":
		handleGetData(request, bc)
	case "tx":
		handleTx(request, bc)
	case "version":
		handleVersion(request, bc)
	default:
		fmt.Println("Unknown command!")
	}

	conn.Close()
}

// sendData sends data to a node
func sendData(addr string, data []byte) {
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		fmt.Printf("%s is not available\n", addr)
		var updatedNodes []string

		for _, node := range KnownNodes {
			if node != addr {
				updatedNodes = append(updatedNodes, node)
			}
		}

		KnownNodes = updatedNodes

		return
	}
	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

// sendVersion 向中心节点发送版本信息
func sendVersion(addr string, bc *blockchain.Blockchain) {
	bestHeight := bc.GetBestHeight()
	payload := Version{
		Version:    1,
		BestHeight: bestHeight,
		AddrFrom:   nodeAddress,
	}
	payloadBytes, err := GobEncode(payload)
	if err != nil {
		log.Panic(err)
	}

	request := append(CommandToBytes("version"), payloadBytes...)
	sendData(addr, request)
}

// SendAddr sends an address to the target node
func SendAddr(address string) {
	nodes := Addr{KnownNodes}
	nodes.AddrList = append(nodes.AddrList, nodeAddress)
	payload, err := GobEncode(nodes)
	if err != nil {
		log.Panic(err)
	}
	request := append(CommandToBytes("addr"), payload...)

	sendData(address, request)
}

// SendBlock sends a block to the target node
func SendBlock(addr string, b *blockchain.Block) {
	data := BlockData{nodeAddress, b.Serialize()}
	payload, err := GobEncode(data)
	if err != nil {
		log.Panic(err)
	}
	request := append(CommandToBytes("block"), payload...)

	sendData(addr, request)
}

// SendInv sends an inventory of blocks or transactions to the target node
func SendInv(address, kind string, items [][]byte) {
	inventory := Inv{nodeAddress, kind, items}
	payload, err := GobEncode(inventory)
	if err != nil {
		log.Panic(err)
	}
	request := append(CommandToBytes("inv"), payload...)

	sendData(address, request)
}

// SendGetBlocks sends a getblocks request to the target node
func SendGetBlocks(address string) {
	payload, err := GobEncode(GetBlocks{nodeAddress})
	if err != nil {
		log.Panic(err)
	}
	request := append(CommandToBytes("getblocks"), payload...)

	sendData(address, request)
}

// SendGetData sends a getdata request to the target node
func SendGetData(address, kind string, id []byte) {
	payload, err := GobEncode(GetData{nodeAddress, kind, id})
	if err != nil {
		log.Panic(err)
	}
	request := append(CommandToBytes("getdata"), payload...)

	sendData(address, request)
}

// SendTx sends a transaction to the target node
func SendTx(addr string, tnx *blockchain.Transaction) {
	data := Tx{nodeAddress, tnx.Serialize()}
	payload, err := GobEncode(data)
	if err != nil {
		log.Panic(err)
	}
	request := append(CommandToBytes("tx"), payload...)

	sendData(addr, request)
}
