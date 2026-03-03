package hash

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
)

// Hash256 表示 32 字节 SHA-256 哈希。
type Hash256 [32]byte

// Sum256 计算 SHA256(data)。
func Sum256(data []byte) Hash256 {
	return sha256.Sum256(data)
}

// DomainHash 计算带域分离的哈希：SHA256(domain || data)。
func DomainHash(domain string, data []byte) Hash256 {
	h := sha256.New()
	h.Write([]byte(domain))
	h.Write(data)
	var out Hash256
	copy(out[:], h.Sum(nil))
	return out
}

// Hash256FromBytes 将 32 字节数组转换为 Hash256。
func Hash256FromBytes(b []byte) (Hash256, error) {
	if len(b) != 32 {
		return Hash256{}, errors.New("Hash256FromBytes 失败：长度必须为 32 字节")
	}
	var h Hash256
	copy(h[:], b)
	return h, nil
}

// Bytes 返回哈希的副本。
func (h Hash256) Bytes() []byte {
	out := make([]byte, 32)
	copy(out, h[:])
	return out
}

// Hex 返回小写 hex。
func (h Hash256) Hex() string {
	return hex.EncodeToString(h[:])
}

func HomeShard(addr20 [20]byte, numShards uint32) (uint32, error) {
	// 计算地址的 home shard：hash(addr20) % numShards
	if numShards == 0 {
		return 0, errors.New("HomeShard失败：numShards 为 0")
	}
	h := Sum256(addr20[:])
	u := binary.BigEndian.Uint64(h[0:8])
	return uint32(u % uint64(numShards)), nil
}

// ParseHash256Hex 从 64 位 hex 字符串解析 Hash256。
func ParseHash256Hex(s string) (Hash256, error) {
	x := strings.ToLower(strings.TrimSpace(s))
	if len(x) != 64 {
		return Hash256{}, errors.New("ParseHash256Hex 失败：长度必须为 64 位 hex")
	}
	b, err := hex.DecodeString(x)
	if err != nil {
		return Hash256{}, errors.New("ParseHash256Hex 失败：hex 非法")
	}
	return Hash256FromBytes(b)
}

func BytesToHex(b []byte) string {
	return hex.EncodeToString(b)
}
