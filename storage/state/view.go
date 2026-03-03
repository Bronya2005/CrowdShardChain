package state

type ShardView struct {
	st *ShardState
	j  *ShardJournal
}

func NewShardView(st *ShardState, j *ShardJournal) *ShardView {
	return &ShardView{
		st: st,
		j:  j,
	}
}

func (v *ShardView) GetAccountRO(addr [20]byte) *account {
	return v.st.GetAccount(addr)
}
func (v *ShardView) GetTaskRO(taskId [32]byte) *Task {
	return v.st.GetTask(taskId)
}

func (v *ShardView) GetAccount(addr [20]byte) *account {
	v.j.TouchAccount(addr)
	return v.st.GetAccount(addr)
}

func (v *ShardView) GetTask(taskId [32]byte) *Task {
	v.j.TouchTask(taskId)
	return v.st.GetTask(taskId)
}
