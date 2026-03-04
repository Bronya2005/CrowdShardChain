package state

import (
	"CrowdShardChain/internal/hash"
	"CrowdShardChain/internal/smt"
)

type ShardState struct {
	Accounts map[[20]byte]*account
	Tasks    map[[32]byte]*Task

	AccountTree *smt.Tree
	TaskTree    *smt.Tree

	StateRoot hash.Hash256
}

func NewShardState() *ShardState {
	return &ShardState{
		Accounts:    make(map[[20]byte]*account),
		Tasks:       make(map[[32]byte]*Task),
		AccountTree: smt.NewTree(),
		TaskTree:    smt.NewTree(),
	}
}
