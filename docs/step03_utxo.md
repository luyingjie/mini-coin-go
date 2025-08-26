# 交易 (Transaction) 和 UTXO (Unspent Transaction Output) 模型

我们将规划如何在这个项目的基础上实现交易（Transaction）和 UTXO（Unspent Transaction Output，未花费交易输出）模型。

这是比特币等许多主流加密货币采用的核心模型，引入它会使我们的迷你区块链项目更加真实和强大。

## 核心概念

### 1. 交易 (Transaction)

*   不再是简单地在区块里存储字符串数据（`block.Data`），而是存储一系列的交易。
*   一笔交易主要由输入 (Inputs) 和输出 (Outputs) 组成。
*   **输入 (TXInput)**: 指明了资金的来源。它会引用之前某笔交易的某个输出 (TXOutput)。可以理解为“花钱”。
*   **输出 (TXOutput)**: 包含了资金的数量（Value）和接收方的地址（ScriptPubKey，锁定脚本）。可以理解为“收钱”。

### 2. UTXO (未花费交易输出)

*   整个区块链中所有还没有被作为输入花掉的输出的集合。
*   一个用户的余额，就是整个 UTXO 集合中，他能够解锁（花费）的所有输出的总和。
*   UTXO 模型没有传统的“账户余额”概念，余额是动态计算出来的。

## 实现方案

我们可以分以下几��步骤来实现：

### 第一步：定义交易结构

我们需要在 `blockchain` 目录下创建新的 `transaction.go` 文件，并在其中定义 `Transaction`, `TXInput`, 和 `TXOutput` 结构体。

*   `TXOutput` 结构:

```go
type TXOutput struct {
    Value      int    // 金额
    ScriptPubKey string // 锁定脚本，这里我们可以简化为接收方的地址
}
```

*   `TXInput` 结构:

```go
type TXInput struct {
    Txid      []byte // 引用来源交易的 ID (哈希)
    Vout      int    // 引用来源交易的某个输出的索引
    ScriptSig string // 解锁脚本，这里我们可以简化为发送方的地址
}
```

*   `Transaction` 结构:

```go
type Transaction struct {
    ID   []byte     // 交易的唯一标识 (哈希)
    Vin  []TXInput  // 交易输入
    Vout []TXOutput // 交易输出
}
```

### 第二步：修改区块结构

我们需要修改 `block.go` 中的 `Block` 结构，让它能够存储交易而不是简单的 `[]byte` 数据。

*   修改 `Block` 结构:

```go
type Block struct {
    Timestamp     int64
    Transactions  []*Transaction // 不再是 Data []byte
    PrevBlockHash []byte
    Hash          []byte
    Nonce         int
}
```

*   修改 `NewBlock` 函数:
    *   它现在应该接收一个交易数组 `[]*Transaction` 作为参数，而不是一个字符串。
    *   计算区块哈希时，需要将所有交易的哈希也包含进去（例如通过构建一个默克尔树 Merkle Tree，或者简单地将所有交易 ID 拼接起来再哈希）。

### 第三步：实现 Coinbase 交易

*   Coinbase 交易是一种特殊的交易，它没有输入。
*   每个区块的第一个交易必须是 Coinbase 交易。它用来给“挖出”这个区块的矿工一笔奖励。
*   我们需要创建一个 `NewCoinbaseTX` 函数来生成这种特殊的交易。它的输入 `TXInput` 的 `Txid` 为空，`Vout` 为 `-1`。

### 第四步：创建和管理 UTXO 集

这是最核心的一步。我们需要一种高效的方式来查找任何地址的 UTXO。直接遍历整个区块链是非常低效的。

*   **方案：建立一个专门的 UTXO 数据库表（Bucket）**
    *   在 `bbolt` 数据库中，除了 `blocks` bucket，我们再创建一个 `utxos` bucket。
    *   这个 `utxos` bucket 的 key 是交易的 ID（Txid），value 是该交易中所有未花费的输出（TXOutput）的序列化列表。
*   **更新 UTXO 集**:
    *   当一个新的区块被添加到链上时，我们需要更新 UTXO 集。
    *   **移除已花费的输出**: 遍历新区块中所有交易的输入 (Inputs)，根据它们引用的 `Txid` 和 `Vout`，去 `utxos` bucket 中���到对应的输出并删除它们（因为它们现在已经被花费了）。
    *   **添加新的未花费输出**: 遍历新区块中所有交易的输出 (Outputs)，将它们作为新的 UTXO 添加到 `utxos` bucket 中。
*   **创建 `UTXOSet` 结构**: 我们可以创建一个 `UTXOSet` 结构体，并为其添加 `FindUTXO(pubkeyHash []byte)` 和 `Update(block *Block)` 等方法来封装对 UTXO 的操作。

### 第五步：实现发送功能和余额查询

*   `NewUTXOTransaction` 函数:
    *   这是创建一笔普通转账交易的核心函数。
    *   它需要 `from`（发送方）、`to`（接收方）、`amount`（金额）作为参数。
    *   **工作流程**:
        1.  使用 `UTXOSet.FindSpendableOutputs(pubkeyHash []byte, amount int)` 找到发送方足够支付 `amount` 的所有 UTXO。
        2.  将这些找到的 UTXO 创建为交易的输入 (Inputs)。
        3.  创建一个或两个输出 (Outputs)：
            *   一个输出是支付给接收方 `to` 的 `amount`。
            *   如果找到的 UTXO 总额大于 `amount`，则需要创建第二个输出，将找零返还给发送方 `from`。
*   `GetBalance` 函数:
    *   这个函数接收一个地址 `address`。
    *   它会调用 `UTXOSet.FindUTXO(pubkeyHash []byte)` 找到该地址所有的 UTXO，然后将它们的 `Value` ���加，就得到了该地址的余额。

### 第六步：更新命令行工具

最后，我们需要更新 `cli.go` 来支持新的功能。

*   **添加新命令**:
    *   `getbalance -address ADDRESS`: 查询某个地址的余额。
    *   `send -from FROM -to TO -amount AMOUNT`: 从一个地址向另一个地址发送指定金额的币。
*   **修改 `addblock` 命令**: 这个命令可以被 `send` 命令间接替代，或者我们可以废弃它，因为现在区块是通过挖矿（发送交易）来创建的。
*   **修改 `printchain` 命令**: 让它能够打印出区块中包含的交易信息。

## 总结执行流程

1.  定义数据结构: `Transaction`, `TXInput`, `TXOutput`。
2.  修改 `Block`: 让其包含 `[]*Transaction`。
3.  实现 `UTXOSet`: 用于管理和更新 UTXO 数据库 bucket。
4.  实现交易创建函数: `NewCoinbaseTX` 和 `NewUTXOTransaction`。
5.  实现余额查询函数: `GetBalance`。
6.  更新 `CLI`: 添加 `send` 和 `getbalance` 命令。