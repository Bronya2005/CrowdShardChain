package state

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
	j.checkpoints = j.checkpoints[:len(j.checkpoints)-1]
	j.depth--

}

func (j *ShardJournal) Rollback() {
	if j.depth == 0 {
		return
	}
	cp := j.checkpoints[len(j.checkpoints)-1]

	// 反向回放 undo
	for i := len(j.undo) - 1; i >= cp; i-- {
		e := j.undo[i]
		switch e.kind {
		case undoAcc:
			if e.hadA {
				j.st.accounts[e.addr] = e.oldA
			} else {
				delete(j.st.accounts, e.addr)
			}
			j.accMark[e.addr] = e.prevMark

		case undoTask:
			if e.hadT {
				j.st.tasks[e.tid] = e.oldT
			} else {
				delete(j.st.tasks, e.tid)
			}
			j.taskMark[e.tid] = e.prevMark
		}
	}

	// 丢弃本层 undo
	j.undo = j.undo[:cp]
	j.checkpoints = j.checkpoints[:len(j.checkpoints)-1]
	j.depth--

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

	cur, ok := j.st.accounts[addr]
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

	cur, ok := j.st.tasks[id]
	if ok && cur != nil {
		e.hadT = true
		e.oldT = cloneTask(cur)
	} else {
		e.hadT = false
		e.oldT = nil
	}

	j.undo = append(j.undo, e)
}
