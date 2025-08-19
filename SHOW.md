证整条链的完整性和有效性。

---

### 项目结构与模块功能

```
mini-coin-go/
├── blockchain/
│   ├── block.go         # 存放 Block 结构体和相关方法
│   ├── blockchain.go    # 存放 Blockchain 结构体和相关方法
│   ├── proofofwork.go   # 存放 ProofOfWork 结构体和相关方法
│   └── utils.go         # 存放一些辅助函数
├── go.mod             # Go 模块文件，用于管理依赖
└── main.go            # 程序主入口，调用 blockchain 包
```

**模块功能说明:**

*   **`blockchain/block.go`**: 定义了 `Block` 结构体，以及创建新区块 (`NewBlock`) 和创世区块 (`NewGenesisBlock`) 的函数。
*   **`blockchain/blockchain.go`**: 定义了 `Blockchain` 结构体，以及创建新链 (`NewBlockchain`) 和添加区块 (`AddBlock`) 的方法。
*   **`blockchain/proofofwork.go`**: 封装了所有关于工作量证明的逻辑，包括 `ProofOfWork` 结构体和其 `Run`, `Validate` 方法。
*   **`blockchain/utils.go`**: 包含项目所需的辅助函数，例如 `IntToHex`。
*   **`main.go`**: 项目的启动入口。它导入 `blockchain` 包，并调用其功能来创建链、添加区块并打印信息，保持了主函数的简洁性。

---

### 如何运行

确保你已经安装了 Go 环境 (Go 1.13+)。

在项目根目录下，打开终端并执行以下命令：

```bash
go run main.go
```

你将会看到程序输出挖矿过程，并最终打印出三个区块（1个创世区块 + 2个新区块）的详细信息。