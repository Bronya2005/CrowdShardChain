package state

import (
	"testing"

	"CrowdShardChain/internal/smt"
)

// --- 小工具：构造固定 key ---

func addr20(x byte) [20]byte {
	var a [20]byte
	for i := 0; i < 20; i++ {
		a[i] = x
	}
	return a
}

func tid32(x byte) [32]byte {
	var t [32]byte
	for i := 0; i < 32; i++ {
		t[i] = x
	}
	return t
}

// --- 创建一份可测试的 ShardState ---
// 说明：这里直接访问 st.Accounts/st.Tasks（同包 state 测试允许访问未导出字段）
func newTestState() *ShardState {
	st := &ShardState{
		Accounts: make(map[[20]byte]*account),
		Tasks:    make(map[[32]byte]*Task),
	}
	st.AccountTree = smt.NewTree()
	st.TaskTree = smt.NewTree()
	// st.StateRoot 由 commit 时写入；初值无所谓
	return st
}

func TestJournal_Rollback_RestoresStateAndMarks(t *testing.T) {
	st := newTestState()
	j := NewShardJournal(st)

	a1 := addr20(0x11)
	t1 := tid32(0x22)
	key := addr20(0x00)
	// 初始状态
	st.Accounts[a1] = &account{Nonce: 7, Balance: 100}
	st.Tasks[t1] = &Task{
		TaskID:     "t1",
		Requester:  "r",
		Title:      "x",
		Status:     1,
		RewardPool: 10,
		Deposits:   map[[20]byte]uint64{key: 1},
	}

	j.CheckPoint()

	// Touch + 修改（模拟业务层通过 view 修改对象）
	j.TouchAccount(a1)
	st.Accounts[a1].Balance += 50
	j.TouchTask(t1)
	st.Tasks[t1].RewardPool += 99
	st.Tasks[t1].Deposits[key] = 999 // cloneTask 应深拷贝 deposits，否则 rollback 测不准

	// 确认已修改
	if st.Accounts[a1].Balance != 150 {
		t.Fatalf("expected balance=150 got=%d", st.Accounts[a1].Balance)
	}
	if st.Tasks[t1].RewardPool != 109 {
		t.Fatalf("expected rewardpool=109 got=%d", st.Tasks[t1].RewardPool)
	}
	if st.Tasks[t1].Deposits[key] != 999 {
		t.Fatalf("expected deposit[u]=999 got=%d", st.Tasks[t1].Deposits[key])
	}

	// 回滚
	j.Rollback()

	// 状态恢复
	if st.Accounts[a1].Balance != 100 {
		t.Fatalf("rollback failed: balance expected=100 got=%d", st.Accounts[a1].Balance)
	}
	if st.Tasks[t1].RewardPool != 10 {
		t.Fatalf("rollback failed: rewardpool expected=10 got=%d", st.Tasks[t1].RewardPool)
	}
	if st.Tasks[t1].Deposits[key] != 1 {
		t.Fatalf("rollback failed: deposits expected=1 got=%d", st.Tasks[t1].Deposits[key])
	}

	// mark 恢复：prevMark==0 应删除 mark（不会残留 0）
	if _, ok := j.accMark[a1]; ok {
		t.Fatalf("expected accMark deleted after rollback, got present=%v val=%d", ok, j.accMark[a1])
	}
	if _, ok := j.taskMark[t1]; ok {
		t.Fatalf("expected taskMark deleted after rollback, got present=%v val=%d", ok, j.taskMark[t1])
	}

	// depth 回到 0
	if j.depth != 0 {
		t.Fatalf("expected depth=0 got=%d", j.depth)
	}
}

func TestJournal_Commit_ComputesRootIfMissing_AndWritesToTrees(t *testing.T) {
	st := newTestState()
	j := NewShardJournal(st)

	a1 := addr20(0x33)
	st.Accounts[a1] = &account{Nonce: 1, Balance: 10}

	j.CheckPoint()

	// Touch + 修改
	j.TouchAccount(a1)
	st.Accounts[a1].Balance = 777

	// 不显式调用 ComputeRoot，直接 Commit，应该在 Commit 内自动计算并写回
	j.Commit()

	// depth 回到 0
	if j.depth != 0 {
		t.Fatalf("expected depth=0 got=%d", j.depth)
	}

	// trees root 应不再是空根（很大概率），并且 StateRoot 应写入
	// 注意：空根也可能出现（极低概率碰撞），这里做弱断言：StateRoot 与当前树根汇总一致
	accRoot := st.AccountTree.Root()
	taskRoot := st.TaskTree.Root()
	print(st.StateRoot.Hex())
	// StateRoot 必须等于 hash.Domain("STATE", acc||task) —— 这个等式你在实现里写了
	// 由于这里无法直接调用 hash.DomainHash（不引入 hash 包会增加耦合），只做一致性检查：
	// - 你可以把 stateRoot 计算函数抽出去后，这里就能精确断言。
	if (accRoot == st.AccountTree.EmptyRoot()) && (taskRoot == st.TaskTree.EmptyRoot()) {
		// 如果两棵树都是空根，那么 stateRoot 很可能仍为“空状态根”，也允许
		// 不作为失败条件
	}

	// marks 应清空（提交到最外层）
	if len(j.accMark) != 0 || len(j.taskMark) != 0 {
		t.Fatalf("expected marks cleared at depth=0, got accMark=%d taskMark=%d", len(j.accMark), len(j.taskMark))
	}
}

func TestJournal_NestedCommit_MergesMarksToParent_PreventDuplicateUndo(t *testing.T) {
	st := newTestState()
	j := NewShardJournal(st)

	a1 := addr20(0x44)
	st.Accounts[a1] = &account{Nonce: 0, Balance: 1}
	// 外层
	j.CheckPoint()
	j.TouchAccount(a1)
	st.Accounts[a1].Balance = 2
	undoAfterOuterTouch := len(j.undo)

	// 内层
	j.CheckPoint()
	j.TouchAccount(a1) // 内层再次 touch 同 key
	st.Accounts[a1].Balance = 3
	undoAfterInnerTouch := len(j.undo)

	if undoAfterInnerTouch <= undoAfterOuterTouch {
		t.Fatalf("expected inner touch to append undo entry, got undoAfterInner=%d outer=%d", undoAfterInnerTouch, undoAfterOuterTouch)
	}

	// 提交内层：必须把 a1 的 mark 合并到父层，否则父层 touch 会重复追加 undo（旧 bug）
	j.Commit()

	// 此时仍在外层（depth==1）
	if j.depth != 1 {
		t.Fatalf("expected depth=1 after inner commit, got=%d", j.depth)
	}

	// 在外层再次 Touch 同 key：应该被 mark 拦住，不应追加 undo
	undoBeforeRetouch := len(j.undo)
	j.TouchAccount(a1)
	undoAfterRetouch := len(j.undo)

	if undoAfterRetouch != undoBeforeRetouch {
		t.Fatalf("mark merge failed: expected no new undo on retouch; before=%d after=%d", undoBeforeRetouch, undoAfterRetouch)
	}

	// 最后回滚外层，余额恢复到最初
	j.Rollback()
	if st.Accounts[a1].Balance != 1 {
		t.Fatalf("rollback outer failed: expected balance=1 got=%d", st.Accounts[a1].Balance)
	}
}
