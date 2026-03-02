package frame

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	magic   = [4]byte{'C', 'S', 'C', '2'}
	version = byte(1)
)

// MaxPayloadBytes 限制 Payload 最大长度，防止 OOM/大包攻击。
const MaxPayloadBytes = 8 << 20 // 8 MiB

// Encode 将 payload 封装为 Frame：
// Magic(4)="CSC2" + Version(1)=1 + Kind(1) + Length(4 BE) + Payload
func Encode(kind Kind, payload []byte) ([]byte, error) {
	// 校验 kind 合法性（避免调用方传入无效 kind）
	if _, err := KindOf(byte(kind)); err != nil {
		return nil, err
	}
	if len(payload) > MaxPayloadBytes {
		return nil, fmt.Errorf("payload 过大：%d > %d", len(payload), MaxPayloadBytes)
	}

	out := make([]byte, 10+len(payload))
	copy(out[0:4], magic[:])
	out[4] = version
	out[5] = byte(kind)
	binary.BigEndian.PutUint32(out[6:10], uint32(len(payload)))
	copy(out[10:], payload)
	return out, nil
}

// Decode 解析 Frame 并返回 (version, kind, payload)。
// 校验：Magic、Version、Kind、Length、最大长度、是否截断/多余。
func Decode(frameBytes []byte) (byte, Kind, []byte, error) {
	if len(frameBytes) < 10 {
		return 0, 0, nil, errors.New("frame 长度不足：< 10")
	}

	// 1) Magic
	if frameBytes[0] != magic[0] || frameBytes[1] != magic[1] || frameBytes[2] != magic[2] || frameBytes[3] != magic[3] {
		return 0, 0, nil, errors.New("magic 不匹配：不是 CSC2 frame")
	}

	// 2) Version
	ver := frameBytes[4]
	if ver != version {
		return 0, 0, nil, fmt.Errorf("不支持的 version：%d", ver)
	}

	// 3) Kind
	k, err := KindOf(frameBytes[5])
	if err != nil {
		return 0, 0, nil, err
	}

	// 4) Length
	n := binary.BigEndian.Uint32(frameBytes[6:10])
	if n > uint32(MaxPayloadBytes) {
		return 0, 0, nil, fmt.Errorf("payload 长度超限：%d > %d", n, MaxPayloadBytes)
	}

	need := 10 + int(n)
	if len(frameBytes) < need {
		return 0, 0, nil, fmt.Errorf("frame 截断：需要 %d 字节，实际 %d 字节", need, len(frameBytes))
	}
	if len(frameBytes) != need {
		return 0, 0, nil, fmt.Errorf("frame 长度不一致：声明 %d 字节 payload，但实际 frame 多出 %d 字节", n, len(frameBytes)-need)
	}

	payload := make([]byte, int(n))
	copy(payload, frameBytes[10:need])
	return ver, k, payload, nil
}
