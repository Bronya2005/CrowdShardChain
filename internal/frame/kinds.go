package frame

import "fmt"

// Kind 表示 Frame 中承载的对象类型。
type Kind uint8

const (
	KindTx      Kind = 1
	KindBlock   Kind = 2
	KindReceipt Kind = 3
	KindMessage Kind = 4
)

// Kind 将输入字节转换为 Kind，并校验是否为已支持类型。
func KindOf(b byte) (Kind, error) {
	k := Kind(b)
	switch k {
	case KindTx, KindBlock, KindReceipt, KindMessage:
		return k, nil
	default:
		return 0, fmt.Errorf("不支持的 Kind：%d", b)
	}
}
