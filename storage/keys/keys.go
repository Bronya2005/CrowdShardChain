package keys

import (
	"encoding/hex"
)

func ShardHex4(shard uint16) string {
	// 固定 4 位小写 hex
	b := []byte{byte(shard >> 8), byte(shard)}
	return hex.EncodeToString(b) // 4 chars
}

// --- st/<shard>/acc/<addr20>
func StAccPrefix(shard uint16) []byte {
	s := "st/" + ShardHex4(shard) + "/acc/"
	return []byte(s)
}

func StAccKey(shard uint16, addr20 [20]byte) []byte {
	p := StAccPrefix(shard)
	k := make([]byte, 0, len(p)+20)
	k = append(k, p...)
	k = append(k, addr20[:]...)
	return k
}

// --- st/<shard>/task/<taskId32>
func StTaskPrefix(shard uint16) []byte {
	s := "st/" + ShardHex4(shard) + "/task/"
	return []byte(s)
}

func StTaskKey(shard uint16, taskId32 [32]byte) []byte {
	p := StTaskPrefix(shard)
	k := make([]byte, 0, len(p)+32)
	k = append(k, p...)
	k = append(k, taskId32[:]...)
	return k
}
