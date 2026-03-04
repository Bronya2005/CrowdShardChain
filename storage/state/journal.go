package state

import (
	"CrowdShardChain/internal/ccj"
	"CrowdShardChain/internal/hash"
	"errors"
)

type undoKind uint8

const (
	undoAcc  undoKind = 1
	undoTask undoKind = 2
)

type undoEntry struct {
	kind  undoKind
	depth uint32

	// account
	addr [20]byte
	oldA *account
	hadA bool

	// task
	tid  [32]byte
	oldT *Task
	hadT bool

	prevMark uint32
}

type ShardJournal struct {
	st *ShardState

	depth       uint32
	checkpoints []int
	undo        []undoEntry

	accMark  map[[20]byte]uint32
	taskMark map[[32]byte]uint32

	rootReady      []bool
	accRootCache   []hash.Hash256
	taskRootCache  []hash.Hash256
	stateRootCache []hash.Hash256
}

func (j *ShardJournal) ensureRootCacheAligned() {
	n := len(j.checkpoints)
	for len(j.rootReady) < n {
		j.rootReady = append(j.rootReady, false)
		j.accRootCache = append(j.accRootCache, hash.Hash256{})
		j.taskRootCache = append(j.taskRootCache, hash.Hash256{})
		j.stateRootCache = append(j.stateRootCache, hash.Hash256{})
	}
}

func (j *ShardJournal) topIndex() int {
	if j.depth == 0 || len(j.checkpoints) == 0 {
		return -1
	}
	return len(j.checkpoints) - 1
}

func NewShardJournal(st *ShardState) *ShardJournal {
	return &ShardJournal{
		st:          st,
		checkpoints: make([]int, 0, 8),
		undo:        make([]undoEntry, 0, 1<<16),
		accMark:     make(map[[20]byte]uint32),
		taskMark:    make(map[[32]byte]uint32),
	}
}

func (j *ShardJournal) CheckPoint() {
	j.depth++
	j.checkpoints = append(j.checkpoints, len(j.undo))
}

func (j *ShardJournal) Commit() {
	if j.depth == 0 {
		return
	}
	j.ensureRootCacheAligned()
	idx := j.topIndex()
	cp := j.checkpoints[idx]
	parentDepth := j.depth - 1

	// 1) 确保本层根已计算并缓存
	if !j.rootReady[idx] {
		if _, err := j.ComputeRoot(); err != nil {
			panic(err)
		}
	}

	// 2) 合并 mark：把本层 touched 的 key 标记为 parentDepth（或清空）
	if parentDepth > 0 {
		for i := cp; i < len(j.undo); i++ {
			e := j.undo[i]
			switch e.kind {
			case undoAcc:
				j.accMark[e.addr] = parentDepth
			case undoTask:
				j.taskMark[e.tid] = parentDepth
			}
		}
	} else {
		// 提交到最外层：不再需要 marks（下一次 Touch 会自动建新 checkpoint）
		for k := range j.accMark {
			delete(j.accMark, k)
		}
		for k := range j.taskMark {
			delete(j.taskMark, k)
		}
		// 可选：也可以清空 undo，避免增长（推荐）
		j.undo = j.undo[:0]
	}

	// 3) 写回树根 + state root（正式生效）
	j.st.AccountTree.SetRoot(j.accRootCache[idx])
	j.st.TaskTree.SetRoot(j.taskRootCache[idx])
	j.st.StateRoot = j.stateRootCache[idx] // 你后续在 state 增加该字段

	// 4) 弹 checkpoint、降 depth
	j.checkpoints = j.checkpoints[:idx]
	j.depth--

	// 5) 弹 root cache 槽
	j.rootReady = j.rootReady[:idx]
	j.accRootCache = j.accRootCache[:idx]
	j.taskRootCache = j.taskRootCache[:idx]
	j.stateRootCache = j.stateRootCache[:idx]
}

func (j *ShardJournal) Rollback() {
	if j.depth == 0 {
		return
	}
	j.ensureRootCacheAligned()
	idx := j.topIndex()      // 本层 index
	cp := j.checkpoints[idx] // 本层 undo 起点

	for i := len(j.undo) - 1; i >= cp; i-- {
		e := j.undo[i]
		switch e.kind {
		case undoAcc:
			if e.hadA {
				j.st.Accounts[e.addr] = e.oldA
			} else {
				delete(j.st.Accounts, e.addr)
			}
			if e.prevMark == 0 {
				delete(j.accMark, e.addr)
			} else {
				j.accMark[e.addr] = e.prevMark
			}

		case undoTask:
			if e.hadT {
				j.st.Tasks[e.tid] = e.oldT
			} else {
				delete(j.st.Tasks, e.tid)
			}
			if e.prevMark == 0 {
				delete(j.taskMark, e.tid)
			} else {
				j.taskMark[e.tid] = e.prevMark
			}
		}
	}

	// 丢弃本层 undo
	j.undo = j.undo[:cp]
	// 弹 checkpoint
	j.checkpoints = j.checkpoints[:idx]
	j.depth--

	// 丢弃本层 root cache 槽
	j.rootReady = j.rootReady[:idx]
	j.accRootCache = j.accRootCache[:idx]
	j.taskRootCache = j.taskRootCache[:idx]
	j.stateRootCache = j.stateRootCache[:idx]
}

func (j *ShardJournal) Clear() {
	j.depth = 0
	j.checkpoints = j.checkpoints[:0]
	j.undo = j.undo[:0]
	for k := range j.accMark {
		delete(j.accMark, k)
	}
	for k := range j.taskMark {
		delete(j.taskMark, k)
	}
	j.rootReady = j.rootReady[:0]
	j.accRootCache = j.accRootCache[:0]
	j.taskRootCache = j.taskRootCache[:0]
	j.stateRootCache = j.stateRootCache[:0]
}

func (j *ShardJournal) TouchAccount(addr [20]byte) {
	if j.depth == 0 {
		j.CheckPoint()
	}
	if j.accMark[addr] == j.depth {
		return
	}

	var e undoEntry
	e.prevMark = j.accMark[addr]
	j.accMark[addr] = j.depth

	e.kind = undoAcc
	e.depth = j.depth
	e.addr = addr

	cur, ok := j.st.Accounts[addr]
	if ok && cur != nil {
		e.hadA = true
		e.oldA = cloneAccount(cur)
	} else {
		e.hadA = false
		e.oldA = nil
	}

	j.undo = append(j.undo, e)
}

func (j *ShardJournal) TouchTask(id [32]byte) {
	if j.depth == 0 {
		j.CheckPoint()
	}
	if j.taskMark[id] == j.depth {
		return
	}

	var e undoEntry
	e.prevMark = j.taskMark[id]
	j.taskMark[id] = j.depth

	e.kind = undoTask
	e.depth = j.depth
	e.tid = id

	cur, ok := j.st.Tasks[id]
	if ok && cur != nil {
		e.hadT = true
		e.oldT = cloneTask(cur)
	} else {
		e.hadT = false
		e.oldT = nil
	}

	j.undo = append(j.undo, e)
}

func (j *ShardJournal) ComputeRoot() (hash.Hash256, error) {
	if j.depth == 0 {
		return hash.Hash256{}, errors.New("journal: no active checkpoint")
	}
	if j.st == nil || j.st.AccountTree == nil || j.st.TaskTree == nil {
		return hash.Hash256{}, errors.New("journal: state trees not initialized")
	}

	j.ensureRootCacheAligned()
	idx := j.topIndex()
	if idx < 0 {
		return hash.Hash256{}, errors.New("journal: invalid checkpoint index")
	}
	if j.rootReady[idx] {
		return j.stateRootCache[idx], nil
	}

	cp := j.checkpoints[idx]

	// 1) 收集本层 touched key（扫描 undo[cp:]，不依赖 mark）
	accTouched := make(map[[20]byte]struct{})
	taskTouched := make(map[[32]byte]struct{})
	for i := cp; i < len(j.undo); i++ {
		e := j.undo[i]
		switch e.kind {
		case undoAcc:
			accTouched[e.addr] = struct{}{}
		case undoTask:
			taskTouched[e.tid] = struct{}{}
		}
	}

	// 2) 从已确认根开始增量计算临时新根（不写回 currentRoot）
	accRoot := j.st.AccountTree.Root()
	for addr := range accTouched {
		smtKey := hash.DomainHash("ACCKEY", addr[:])

		a, ok := j.st.Accounts[addr]
		if !ok || a == nil {
			var err error
			accRoot, err = j.st.AccountTree.DeleteFrom(accRoot, smtKey)
			if err != nil {
				return hash.Hash256{}, err
			}
			continue
		}
		val, err := ccj.Marshal(a)
		if err != nil {
			return hash.Hash256{}, err
		}
		buf := make([]byte, 0, 20+len(val))
		buf = append(buf, addr[:]...)
		buf = append(buf, val...)
		leaf := hash.DomainHash("ACC", buf)

		accRoot, err = j.st.AccountTree.UpdateFrom(accRoot, smtKey, leaf)
		if err != nil {
			return hash.Hash256{}, err
		}
	}

	taskRoot := j.st.TaskTree.Root()
	for tid := range taskTouched {
		smtKey := hash.DomainHash("TASKKEY", tid[:])

		t, ok := j.st.Tasks[tid]
		if !ok || t == nil {
			var err error
			taskRoot, err = j.st.TaskTree.DeleteFrom(taskRoot, smtKey)
			if err != nil {
				return hash.Hash256{}, err
			}
			continue
		}
		val, err := ccj.Marshal(t)
		if err != nil {
			return hash.Hash256{}, err
		}
		buf := make([]byte, 0, 32+len(val))
		buf = append(buf, tid[:]...)
		buf = append(buf, val...)
		leaf := hash.DomainHash("TASK", buf)

		taskRoot, err = j.st.TaskTree.UpdateFrom(taskRoot, smtKey, leaf)
		if err != nil {
			return hash.Hash256{}, err
		}
	}

	sb := make([]byte, 0, 64)
	sb = append(sb, accRoot[:]...)
	sb = append(sb, taskRoot[:]...)
	stateRoot := hash.DomainHash("STATE", sb)

	j.accRootCache[idx] = accRoot
	j.taskRootCache[idx] = taskRoot
	j.stateRootCache[idx] = stateRoot
	j.rootReady[idx] = true
	return stateRoot, nil
}
