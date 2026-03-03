package state

import (
	"CrowdShardChain/internal/ccj"
	"CrowdShardChain/internal/hash"
	"CrowdShardChain/internal/merkle"
	"bytes"
	"sort"
)

type leafItem struct {
	sortKey []byte
	leaf    hash.Hash256
}

func (s *ShardState) StateRoot() (hash.Hash256, error) {
	items := make([]leafItem, 0, len(s.accounts)+len(s.tasks))
	leaves := make([]hash.Hash256, 0, len(s.accounts)+len(s.tasks))
	for k, a := range s.accounts {
		snap, err := EncodeAccount(k, a)
		if err != nil {
			return hash.Hash256{}, err
		}
		val, err := ccj.Marshal(snap)
		if err != nil {
			return hash.Hash256{}, err
		}
		sk := make([]byte, 1+20)
		sk[0] = 0x01
		copy(sk[1:], k[:])
		items = append(items, leafItem{
			sortKey: sk,
			leaf:    StateLeafHash(0x01, k[:], val),
		})
	}
	for k, t := range s.tasks {
		snap, err := EncodeTask(k, t)
		if err != nil {
			return hash.Hash256{}, err
		}
		val, err := ccj.Marshal(snap)
		if err != nil {
			return hash.Hash256{}, err
		}
		sk := make([]byte, 1+32)
		sk[0] = 0x02
		copy(sk[1:], k[:])
		items = append(items, leafItem{
			sortKey: sk,
			leaf:    StateLeafHash(0x02, k[:], val),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return bytes.Compare(items[i].sortKey, items[j].sortKey) < 0
	})
	for _, item := range items {
		leaves = append(leaves, item.leaf)
	}
	return merkle.Root(leaves), nil
}

func StateLeafHash(kind byte, key []byte, value []byte) hash.Hash256 {
	buf := make([]byte, 0, 1+len(key)+len(value))
	buf = append(buf, kind)
	buf = append(buf, key...)
	buf = append(buf, value...)
	return hash.DomainHash("STATE", buf)
}
