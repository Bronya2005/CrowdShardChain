# CCJ（Canonical Consensus JSON）

CCJ 用于 CrowdShardChain 的共识级确定性 JSON 编码，保证跨语言/跨实现的哈希与签名输入一致。

## 接口

- `Marshal(v any) ([]byte, error)`
  - 将 Go 值编码为 canonical JSON（CCJ）。
  - 适用于 TxCore、BlockHeader、Message、Receipt 等参与 TxID/Merkle/签名的对象。

- `Unmarshal(data []byte, out any) error`
  - 严格解析 CCJ JSON，并将结果注入到 `out`。

## 规范

1. 允许的 JSON 类型：
   - `null`, `boolean`, `string`, `integer`, `array`, `object`
   - 禁止：浮点、小数、科学计数法、NaN/Infinity。

2. 整数规则（十进制）：
   - 仅允许 `-?[0-9]+`
   - 禁止前导 `+`
   - 禁止前导 `0`（除 `0` 本身）
   - 禁止 `-0`
   - 必须在 `int64` 范围内。

3. 对象（object）规则：
   - 键必须是字符串
   - 禁止重复键
   - canonical 输出时，键按 UTF-8 字节序升序排序。

4. 输出格式（canonical）：
   - 输出不包含任何空白字符（无空格/换行/缩进）
   - 字符串采用最小必要转义：
     - 必须转义：`\"`, `\\`
     - 控制字符（0x00-0x1F）必须转义：`\n`, `\r`, `\t` 或 `\u00XX`。

5. 时间戳：
   - CCJ 不定义时间类型。时间字段应在上层使用 `int64`（Unix 时间戳，单位由上层规范固定）。

6. Go 类型约束（Marshal/Unmarshal）：
   - 支持：`nil/bool/string/(u)int`、`slice/array`、`map[string]T`、`struct(json tag + omitempty)`
   - 不支持：`float32/float64`、`[]byte`、`[N]byte`。