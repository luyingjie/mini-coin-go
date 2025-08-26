# Mini-Coin-Go: 一个简单的 Go 区块链学习项目

这是一个使用 Go 语言构建的极简区块链实现。项目通过 BoltDB 实现了数据持久化，并提供了一个命令行界面 (CLI) 用于与区块链进行交互。

---

### 核心特性

- **区块链核心**: 实现了区块、链、哈希和工作量证明 (Proof of Work) 等核心概念。
- **数据持久化**: 使用嵌入式键值数据库 BoltDB (`bbolt`) 将区块链数据存储在本地文件 (`blockchain.db`) 中，确保了数据的持久性。
- **命令行界面 (CLI)**: 提供了一个简单的命令行工具来与区块链交互，如添加新区块和打印整条链。

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

通过命令行��区块链进行交互。

**打印区块链:**

使用 `printchain` 命令来显示链上的所有区块。

```bash
# Windows
.\mini-coin-go.exe printchain

# macOS / Linux
./mini-coin-go printchain
```

**添加新区块:**

使用 `addblock` 命令并附带 `-data` 标志来向链上添加一个新区块。

```bash
# Windows
.\mini-coin-go.exe addblock -data "这里是你要存储的数据"

# macOS / Linux
./mini-coin-go addblock -data "这里是你要存储的数据"
```

**示例:**

```bash
# 添加第一个区块
.\mini-coin-go.exe addblock -data "Send 1 BTC to Ivan"

# 添加第二个区块
.\mini-coin-go.exe addblock -data "Send 2 more BTC to Ivan"

# 查看结果
.\mini-coin-go.exe printchain
```

---

### 项目结构

```
mini-coin-go/
├── blockchain/
│   ├── block.go         # Block 结构体, 序列化/反序列化
│   ├── blockchain.go    # Blockchain 结构体, 数据库交互, 迭代器
│   ├── proofofwork.go   # 工作量证明逻辑
│   └── utils.go         # 辅助函数
├── cli.go             # 命令行界面逻辑
├── go.mod             # Go 模块文件
├── go.sum             # 依赖项校验和
├── main.go            # 程序主入口，调用 CLI
└── blockchain.db      # (自动生成) 区块链数据库文件
```
