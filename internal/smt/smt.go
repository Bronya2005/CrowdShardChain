package smt

import (
	"crypto/sha256"
	"errors"

	"CrowdShardChain/internal/ccj"
	"CrowdShardChain/internal/hash"
)

// SMT（Sparse Merkle Tree）—— 256 层固定高度
//
// 规则：
// - 空叶/空子树：EmptyLeaf = H(0x00)
// - 父节点：NodeHash = H(0x01 || left || right)
// - 叶子：由调用者直接提供 leafHash（32 bytes），SMT 不对 value 做编码/域分离。
//   - Update(key, leafHash) 把该 key 的叶子设为 leafHash
//   - Delete(key) 把叶子设为空叶
//
// H 使用 SHA256；如你想统一到 internal/hash 的实现，也可以替换。
type Tree struct {
	store         map[hash.Hash256]Node
	defaultHashes [257]hash.Hash256
	currentRoot   hash.Hash256
}

type Node struct {
	Left  hash.Hash256 `json:"left"`
	Right hash.Hash256 `json:"right"`
}

func NewTree() *Tree {
	t := &Tree{
		store: make(map[hash.Hash256]Node, 1024),
	}
	t.initDefaults()
	t.currentRoot = t.defaultHashes[0]
	return t
}

// H = SHA256(bytes)
func H(b []byte) hash.Hash256 {
	sum := sha256.Sum256(b)
	return hash.Hash256(sum)
}

func (t *Tree) initDefaults() {
	// 空叶：H(0x00)
	t.defaultHashes[256] = H([]byte{0x00})

	// 空子树根：自底向上 H(0x01||child||child)
	for d := 255; d >= 0; d-- {
		child := t.defaultHashes[d+1]
		t.defaultHashes[d] = nodeHash(child, child)
	}
}

func nodeHash(left, right hash.Hash256) hash.Hash256 {
	buf := make([]byte, 0, 1+32+32)
	buf = append(buf, 0x01)
	buf = append(buf, left[:]...)
	buf = append(buf, right[:]...)
	return H(buf)
}

// Root 返回当前树的根（可直接写入区块头/快照）
func (t *Tree) Root() hash.Hash256 { return t.currentRoot }

// SetRoot 用于切换当前树的“工作根”（例如回放到历史 root）
// 注意：如果 store 里没有对应 root 的节点，将无法生成 proof，但 root 仍可用于承诺。
func (t *Tree) SetRoot(root hash.Hash256) { t.currentRoot = root }

func (t *Tree) EmptyRoot() hash.Hash256 { return t.defaultHashes[0] }
func (t *Tree) EmptyLeaf() hash.Hash256 { return t.defaultHashes[256] }

// getBit：取 key 的第 depth 位（0..255），从高位到低位。
func getBit(key hash.Hash256, depth int) int {
	b := key[depth/8]
	shift := 7 - (depth % 8)
	if ((b >> shift) & 1) == 1 {
		return 1
	}
	return 0
}

func (t *Tree) isDefault(h hash.Hash256, depth int) bool {
	return h == t.defaultHashes[depth]
}

func (t *Tree) loadNode(h hash.Hash256, depth int) (Node, bool) {
	// 默认空子树不存储
	if t.isDefault(h, depth) {
		return Node{}, false
	}
	n, ok := t.store[h]
	return n, ok
}

// ------------------- 更新接口（省略 root 参数） -------------------

// Update：基于当前 Root 更新，并写回当前 Root
func (t *Tree) Update(key hash.Hash256, leafHash hash.Hash256) (hash.Hash256, error) {
	newRoot, err := t.UpdateFrom(t.currentRoot, key, leafHash)
	if err != nil {
		return hash.Hash256{}, err
	}
	t.currentRoot = newRoot
	return newRoot, nil
}

// Delete：基于当前 Root 删除，并写回当前 Root
func (t *Tree) Delete(key hash.Hash256) (hash.Hash256, error) {
	newRoot, err := t.DeleteFrom(t.currentRoot, key)
	if err != nil {
		return hash.Hash256{}, err
	}
	t.currentRoot = newRoot
	return newRoot, nil
}

// ------------------- 版本化接口（保留 root 参数） -------------------

// UpdateFrom：从指定 root 版本更新，返回新 root（不自动写回 currentRoot）
func (t *Tree) UpdateFrom(root hash.Hash256, key hash.Hash256, leafHash hash.Hash256) (hash.Hash256, error) {
	return t.updateRec(root, 0, key, &leafHash)
}

// DeleteFrom：从指定 root 版本删除，返回新 root（不自动写回 currentRoot）
func (t *Tree) DeleteFrom(root hash.Hash256, key hash.Hash256) (hash.Hash256, error) {
	return t.updateRec(root, 0, key, nil)
}

// updateRec：递归更新 depth=0..256
// - depth==256：val==nil -> EmptyLeaf；否则 -> *val（直接作为 leaf hash）
func (t *Tree) updateRec(nodeH hash.Hash256, depth int, key hash.Hash256, val *hash.Hash256) (hash.Hash256, error) {
	if depth == 256 {
		if val == nil {
			return t.defaultHashes[256], nil
		}
		return *val, nil
	}

	var left, right hash.Hash256
	n, ok := t.loadNode(nodeH, depth)
	if ok {
		left, right = n.Left, n.Right
	} else {
		left = t.defaultHashes[depth+1]
		right = t.defaultHashes[depth+1]
	}

	if getBit(key, depth) == 0 {
		newLeft, err := t.updateRec(left, depth+1, key, val)
		if err != nil {
			return hash.Hash256{}, err
		}
		left = newLeft
	} else {
		newRight, err := t.updateRec(right, depth+1, key, val)
		if err != nil {
			return hash.Hash256{}, err
		}
		right = newRight
	}

	newH := nodeHash(left, right)

	// 子树回到默认空子树：不存节点
	if newH == t.defaultHashes[depth] {
		return newH, nil
	}

	// 内容寻址存储：nodeHash -> children
	t.store[newH] = Node{Left: left, Right: right}
	return newH, nil
}

// ------------------- Proof -------------------

type Proof struct {
	Siblings [256]hash.Hash256 `json:"siblings"`
}

// Prove：对当前 Root 生成证明
func (t *Tree) Prove(key hash.Hash256) (Proof, error) {
	return t.ProveFrom(t.currentRoot, key)
}

// ProveFrom：对指定 root 生成证明
func (t *Tree) ProveFrom(root hash.Hash256, key hash.Hash256) (Proof, error) {
	var p Proof
	cur := root

	for depth := 0; depth < 256; depth++ {
		var left, right hash.Hash256
		n, ok := t.loadNode(cur, depth)
		if ok {
			left, right = n.Left, n.Right
		} else {
			left = t.defaultHashes[depth+1]
			right = t.defaultHashes[depth+1]
		}

		if getBit(key, depth) == 0 {
			p.Siblings[depth] = right
			cur = left
		} else {
			p.Siblings[depth] = left
			cur = right
		}
	}
	return p, nil
}

// VerifyProof：验证 (key, leafHash/空叶, proof) 能否推出 root
func (t *Tree) VerifyProof(root hash.Hash256, key hash.Hash256, leafHash hash.Hash256, present bool, proof Proof) bool {
	cur := t.defaultHashes[256]
	if present {
		cur = leafHash
	}
	for depth := 255; depth >= 0; depth-- {
		sib := proof.Siblings[depth]
		if getBit(key, depth) == 0 {
			cur = nodeHash(cur, sib)
		} else {
			cur = nodeHash(sib, cur)
		}
	}
	return cur == root
}

// CheckRootKnown：若你要生成 proof，root 至少应该可从 store 走到子节点（或是空根）
func (t *Tree) CheckRootKnown(root hash.Hash256) error {
	if root == t.defaultHashes[0] {
		return nil
	}
	if _, ok := t.store[root]; ok {
		return nil
	}
	return errors.New("smt: root not found in store (proof generation may be impossible)")
}

// ------------------- 存储：CCJ 编码/解码 -------------------

// Snapshot 用于持久化 SMT（演示/测试/简单落盘）
// 注意：把整个 store 编码会比较大；真实工程通常把 store 放 KV DB，只存 root。
type Snapshot struct {
	Root  hash.Hash256          `json:"root"`
	Nodes map[hash.Hash256]Node `json:"nodes"`
}

// MarshalSnapshot：把 (root + node store) 编码为 CCJ bytes
func (t *Tree) Marshal() ([]byte, error) {
	snap := Snapshot{
		Root:  t.currentRoot,
		Nodes: t.store,
	}
	return ccj.Marshal(snap)
}

// UnmarshalSnapshot：从 CCJ bytes 恢复 SMT（root + store）
// - 会重新初始化 defaultHashes（保证规则一致）
func Unmarshal(b []byte) (*Tree, error) {
	var snap Snapshot
	if err := ccj.Unmarshal(b, &snap); err != nil {
		return nil, err
	}

	t := &Tree{
		store: make(map[hash.Hash256]Node, len(snap.Nodes)),
	}
	t.initDefaults()
	for k, v := range snap.Nodes {
		t.store[k] = v
	}
	t.currentRoot = snap.Root
	return t, nil
}
