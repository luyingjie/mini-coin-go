package network

// Version 消息，用于节点间同步区块链高度
type Version struct {
	Version    int
	BestHeight int
	AddrFrom   string
}

// GetBlocks 消息，用于向其他节点请求区块哈希列表
type GetBlocks struct {
	AddrFrom string
}

// Inv 消息，用于告诉其他节点自己拥有的区块或交易信息
type Inv struct {
	AddrFrom string
	Type     string
	Items    [][]byte
}

// GetData 消息，用于根据哈希请求具体的区块或交易数据
type GetData struct {
	AddrFrom string
	Type     string
	ID       []byte
}

// BlockData 消息，用于发送一个完整的区块数据
type BlockData struct {
	AddrFrom string
	Block    []byte
}

// Tx 消息，用于发送一个交易数据
type Tx struct {
	AddrFrom    string
	Transaction []byte
}

// Addr 消息，用于在节点间共享和广播其他节点的地址
type Addr struct {
	AddrList []string
}
