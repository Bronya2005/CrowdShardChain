package merkle

import (
	"errors"

	"CrowdShardChain/internal/hash"
)

// ProofItem 表示一层的兄弟节点及其左右位置。
// Left=true 表示 sibling 在左侧：parent = H(0x01 || sibling || cur)
// Left=false 表示 sibling 在右侧：parent = H(0x01 || cur || sibling)
type ProofItem struct {
	Sibling hash.Hash256
	Left    bool
}

// BuildProof 为 leaves[index] 构造 Merkle Proof。
func BuildProof(leaves []hash.Hash256, index int) ([]ProofItem, error) {
	if len(leaves) == 0 {
		return nil, errors.New("构造Merkle证明失败：叶子为空")
	}
	if index < 0 || index >= len(leaves) {
		return nil, errors.New("构造Merkle证明失败：index越界")
	}

	// 当前层节点
	level := make([]hash.Hash256, len(leaves))
	copy(level, leaves)

	proof := make([]ProofItem, 0, 64)
	idx := index

	for len(level) > 1 {
		// 奇数复制最后一个
		if len(level)%2 == 1 {
			level = append(level, level[len(level)-1])
		}

		// 兄弟索引
		var sib int
		if idx%2 == 0 {
			sib = idx + 1
			proof = append(proof, ProofItem{Sibling: level[sib], Left: false})
		} else {
			sib = idx - 1
			proof = append(proof, ProofItem{Sibling: level[sib], Left: true})
		}

		// 上升一层
		next := make([]hash.Hash256, 0, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			next = append(next, parent(level[i], level[i+1]))
		}
		level = next
		idx = idx / 2
	}

	return proof, nil
}

// VerifyProof 校验：给定 leaf、proof、期望 root，是否成立。
func VerifyProof(leaf hash.Hash256, proof []ProofItem, root hash.Hash256) (bool, error) {
	cur := leaf
	for _, it := range proof {
		if it.Left {
			cur = parent(it.Sibling, cur)
		} else {
			cur = parent(cur, it.Sibling)
		}
	}
	return cur == root, nil
}
