# Day 03: 实现交易与 UTXO 模型

## 目标

本节的目标是在 Mini-Coin-Go 项目中引入交易（Transaction）和 UTXO（Unspent Transaction Output）模型，使其更接近真实的区块链实现。这将允许我们进行币的发送和接收，并查询地址余额。

## 方案

我们严格遵循了项目根目录下 `utxo.md` 文件中定义的方案。主要步骤包括：

1.  **定义交易结构**：创建 `Transaction`、`TXInput` 和 `TXOutput` 结构体。
2.  **修改区块结构**：更新 `Block` 结构体以包含交易列表，并调整区块哈希的计算方式。
3.  **实现 Coinbase 交易**：创建一种特殊的交易，用于奖励矿工。
4.  **创建和管理 UTXO 集**：实现 UTXO 数据库表，并提供更新和查询 UTXO 的功能。
5.  **实现发送功能和余额查询**：编写创建普通交易和查询地址余额的逻辑。
6.  **更新命令行工具**：扩展 CLI 以支持新的交易和钱包功能。

## 实施过程

### 1. 定义交易结构 (`blockchain/transaction.go`)

创建 `blockchain/transaction.go` 文件，并定义 `TXOutput`、`TXInput` 和 `Transaction` 结构体。同时，为 `Transaction` 添加 `Hash()` 方法用于计算交易 ID，并为 `TXOutput` 添加 `IsLockedWithKey()` 方法用于验证输出是否被指定公钥哈希锁定。此外，还添加了 `IsCoinbase()` 方法来判断是否为 Coinbase 交易，以及 `TXOutputs` 结构体及其序列化/反序列化方法，用于 UTXO 集合的存储。

### 2. 修改区块结构 (`blockchain/block.go` 和 `blockchain/merkle.go`)

修改 `blockchain/block.go`：
*   将 `Block` 结构体中的 `Data []byte` 字段替换为 `Transactions []*Transaction`。
*   修改 `NewBlock` 函数，使其接收 `[]*Transaction` 作为参数。
*   添加 `HashTransactions()` 方法，用于计算区块中所有交易的 Merkle 根哈希，并将其纳入区块哈希的计算中。

创建 `blockchain/merkle.go` 文件，实现 `MerkleTree` 和 `MerkleNode` 结构体，以及 `NewMerkleTree` 和 `NewMerkleNode` 函数，用于构建和计算 Merkle 树的根哈希。

### 3. 实现 Coinbase 交易 (`blockchain/transaction.go`)

在 `blockchain/transaction.go` 中添加 `NewCoinbaseTX` 函数，用于创建 Coinbase 交易。这种交易没有输入，只有一个输出，用于奖励矿工。

### 4. 创建和管理 UTXO 集 (`blockchain/blockchain.go`)

修改 `blockchain/blockchain.go`：
*   在 `NewBlockchain` 函数中，初始化 `UTXOSet` 并调用 `Reindex()` 方法来构建初���的 UTXO 集。
*   添加 `UTXOSet` 结构体，包含 `Blockchain` 的引用。
*   为 `UTXOSet` 添加 `FindSpendableOutputs()` 方法，用于查找指定金额的可用 UTXO。
*   为 `UTXOSet` 添加 `FindUTXO()` 方法，用于查找指定地址的所有 UTXO。
*   为 `UTXOSet` 添加 `Reindex()` 方法，用于重建 UTXO 集。
*   为 `UTXOSet` 添加 `Update()` 方法，用于在添加新区块时更新 UTXO 集（移除已花费的输出，添加新的未花费输出）。
*   修改 `MineBlock` (原 `AddBlock`) 函数，使其接收 `[]*Transaction` 作为参数。
*   添加 `FindUTXO()` 方法到 `Blockchain` 结构体中，用于遍历区块链并找出所有未花费的交易输出。

### 5. 实现发送功能和余额查询 (`blockchain/blockchain.go`)

在 `blockchain/blockchain.go` 中添加 `NewUTXOTransaction` 函数，用于创建一笔普通转账交易。该函数会查找发送方足够的 UTXO，创建交易输入和输出，并处理找零。

### 6. 更新命令行工具 (`cli.go`, `main.go`, `wallet/`, `blockchain/utils.go`)

修改 `cli.go`：
*   更新 `printUsage` 函数，添加新命令的说明。
*   修改 `Run` 函数，解析并处理新的命令行参数。
*   删除旧的 `addBlock` 方法。
*   添加 `createBlockchain` 方法，用于创建区块链。
*   添加 `createWallet` 方��，用于生成新钱包。
*   添加 `getBalance` 方法，用于查询地址余额。
*   添加 `listAddresses` 方法，用于列出所有钱包地址。
*   修改 `printChain` 方法，使其能够打印区块中的交易详情。
*   添加 `send` 方法，用于发送交易。

修改 `main.go`：
*   更新 `main` 函数，移除直接对 `blockchain` 的引用，改为通过 `CLI` 结构体进行操作。

创建 `wallet` 目录，并在其中创建 `wallet/wallet.go` 和 `wallet/wallets.go`：
*   `wallet/wallet.go`：定义 `Wallet` 结构体，包含私钥和公钥，并实现密钥对生成、地址生成等功能。
*   `wallet/wallets.go`：定义 `Wallets` 结构体，用于管理多个钱包的集合，并实现钱包的创建、加载和保存到文件等功能。

修改 `blockchain/utils.go`：
*   添加 `Base58Encode` 和 `Base58Decode` 函数，用于 Base58 编码和解码。
*   添加 `ValidateAddress` 函数，用于验证地址的有效性。
*   添加 `checksum` 函数，用于生成地址校验和。
*   添加 `HashPubKey` 函数，用于哈希公钥。

## 总结

通过以上步骤，我们成功地在 Mini-Coin-Go 项目中实现了交易和 UTXO 模型。现在，我们的区块链能够处理更复杂的价值转移逻辑，并为后续的签名和验证功能奠定了基础。
