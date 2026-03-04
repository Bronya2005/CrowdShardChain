package hash

import (
	"encoding/hex"
	"errors"
)

func BytesToHex(b []byte) string {
	return hex.EncodeToString(b)
}

func HexToBytes(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, errors.New("hex 长度必须为偶数")
	}
	// 严格小写校验
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		return nil, errors.New("hex 非法字符：仅允许小写 0-9a-f")
	}
	// DecodeString 会分配新切片并返回
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, errors.New("hex 解码失败：" + err.Error())
	}
	return b, nil
}
