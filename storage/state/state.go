package state

type ShardState struct {
	accounts map[[20]byte]*account
	tasks    map[[32]byte]*Task
}

func NewShardState() *ShardState {
	return &ShardState{
		accounts: make(map[[20]byte]*account),
		tasks:    make(map[[32]byte]*Task),
	}
}
