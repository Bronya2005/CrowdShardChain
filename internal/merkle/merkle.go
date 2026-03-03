package merkle

import (
	"CrowdShardChain/internal/hash"
)

// Merkle 规则：
// - 输入：叶子哈希列表（Hash256）
// - 空集合：Root = H(0x00)
// - 内部节点：H(0x01 || left || right)
// - 节点数为奇数：复制最后一个节点参与配对

var (
	emptyPrefix = []byte{0x00}
	nodePrefix  = []byte{0x01}
)

// Root 计算叶子集合的 Merkle Root。
func Root(leaves []hash.Hash256) hash.Hash256 {
	if len(leaves) == 0 {
		return hash.Sum256(emptyPrefix)
	}

	level := make([]hash.Hash256, len(leaves))
	copy(level, leaves)

	for len(level) > 1 {
		// 若为奇数，复制最后一个
		if len(level)%2 == 1 {
			level = append(level, level[len(level)-1])
		}

		next := make([]hash.Hash256, 0, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			next = append(next, parent(level[i], level[i+1]))
		}
		level = next
	}
	return level[0]
}

// parent 计算父节点哈希：H(0x01 || left || right)
func parent(left, right hash.Hash256) hash.Hash256 {
	buf := make([]byte, 0, 1+32+32)
	buf = append(buf, nodePrefix...)
	buf = append(buf, left[:]...)
	buf = append(buf, right[:]...)
	return hash.Sum256(buf)
}
