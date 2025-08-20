# Mini-Coin-Go: 一个简单的 Go 区块链学习项目

这是一个使用 Go 语言构建的极简区块链实现。项目通过 BoltDB 实现了数据持久化，并提供了一个命令行界面 (CLI) 用于与区块链进行交互。

---

### 核心特性

- **区块链核心**: 实现了区块、链、哈希和工作量证明 (Proof of Work) 等核心概念。
- **交易与 UTXO**: 引入了交易（Transaction）和未花费交易输出（UTXO）模型，支持转账和余额查询。
- **数据持久化**: 使用嵌入式键值数据库 BoltDB (`bbolt`) 将区块链数据存储在本地文件 (`blockchain.db`) 中，确保了数据的持久性。
- **命令行界面 (CLI)**: 提供了一个简单的命令行工具来与区块链交互，如创建钱包、查询余额、发送交易和打印整条链。

---

### 如何运行

#### 1. 环境准备

确保你已经安装了 Go 环境 (Go 1.13+)。

#### 2. 构建项目

在项目根目录下，打开终端并执行以下命令来编译生成可执行文件：

```bash
go build
```

这将会生成一个名为 `mini-coin-go.exe` (Windows) 或 `mini-coin-go` (macOS/Linux) 的文件。

#### 3. 使用方法

通过��令行与区块链进行交互。

**创建区块链:**

使用 `createblockchain` 命令来创建新的区块链，并指定接收创世区块奖励的地址。

```bash
# Windows
.\mini-coin-go.exe createblockchain -address YOUR_ADDRESS

# macOS / Linux
./mini-coin-go createblockchain -address YOUR_ADDRESS
```

**创建钱包:**

使用 `createwallet` 命令来生成一个新的钱包地址。

```bash
# Windows
.\mini-coin-go.exe createwallet

# macOS / Linux
./mini-coin-go createwallet
```

**查询余额:**

使用 `getbalance` 命令来查询某个地址的余额。

```bash
# Windows
.\mini-coin-go.exe getbalance -address YOUR_ADDRESS

# macOS / Linux
./mini-coin-go getbalance -address YOUR_ADDRESS
```

**列出所有地址:**

使用 `listaddresses` 命令来列出所有已创建的钱包地址。

```bash
# Windows
.\mini-coin-go.exe listaddresses

# macOS / Linux
./mini-coin-go listaddresses
```

**发送交易:**

使用 `send` 命令来从一个地址向另一个地址发送指定金额的币。

```bash
# Windows
.\mini-coin-go.exe send -from FROM_ADDRESS -to TO_ADDRESS -amount AMOUNT

# macOS / Linux
./mini-coin-go send -from FROM_ADDRESS -to TO_ADDRESS -amount AMOUNT
```

**打印区块链:**

使用 `printchain` 命令来显示链上的所有区块和交易详情。

```bash
# Windows
.\mini-coin-go.exe printchain

# macOS / Linux
./mini-coin-go printchain
```

---

### 项目结构

```
mini-coin-go/
├── blockchain/
│   ├── block.go         # Block 结构体, 序列化/反序列化
│   ├── blockchain.go    # Blockchain 结构体, 数据库交互, 迭代器, UTXO 管理
│   ├── merkle.go        # Merkle Tree 实现
│   ├── proofofwork.go   # 工作量证明逻辑
│   ├── transaction.go   # 交易 (Transaction), 输入 (TXInput), 输出 (TXOutput) 结构体
│   └── utils.go         # 辅助函数 (哈希, Base58 编码/解码, 地址验证)
├── wallet/
│   ├── wallet.go        # 钱包 (Wallet) 结构体, 密钥生成, 地址生成
│   └── wallets.go       # 钱包集合管理, 钱包文件存储
├── cli.go             # 命令行界面逻辑
├── go.mod             # Go 模块文件
├── go.sum             # 依赖项校验和
├── main.go            # 程序主入口，调用 CLI
└── blockchain.db      # (自动生成) 区块链数据库文件
```