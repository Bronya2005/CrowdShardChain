# Frame（CSC2）

Frame 是 CrowdShardChain 的二进制封装格式，用于在网络传输/落盘存储时承载链对象（Tx/Block/Receipt/Message），提供类型区分、长度边界与版本控制。

## 格式

- Magic：固定 `CSC2`（4 字节）
- Version：当前固定 `1`（1 字节）
- Kind：对象类型（1 字节）
- Length：Payload 字节长度（uint32，大端，4 字节）
- Payload：`Length` 字节，为 CCJ canonical JSON bytes

头部固定长度：10 字节。

## Kind 定义

- `1` = Tx
- `2` = Block
- `3` = Receipt
- `4` = Message

## 接口

包 `internal/frame` 提供以下接口：

- `Encode(kind Kind, payload []byte) ([]byte, error)`
  - 将 `payload` 封装为 Frame。
  - 校验：Kind 合法、payload 长度不超限。

- `Decode(frameBytes []byte) (version byte, kind Kind, payload []byte, err error)`
  - 解析 Frame 并返回版本、Kind、Payload。
  - 校验：Magic、Version、Kind、Length、是否截断、是否有多余字节。

- `KindOf(b byte) (Kind, error)`
  - 将字节转换为 Kind，并校验是否为已支持类型。