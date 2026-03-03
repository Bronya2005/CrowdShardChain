package tx

import (
	"CrowdShardChain/internal/ccj"
	"CrowdShardChain/internal/hash"
	"errors"
)

func (tx *Tx) ForceShard(numShards uint32) error {
	if tx == nil {
		return errors.New("交易不能为空")
	}

	homeShard, err := hash.HomeShard(tx.From, numShards)
	if err != nil {
		return err
	}
	tx.ShardID = homeShard
	return nil
}

func (tx *Tx) ComputeTxID() (hash.Hash256, error) {
	core := tx.Core()
	coreBytes, err := ccj.Marshal(core)
	if err != nil {
		return hash.Hash256{}, err
	}
	return hash.DomainHash("TXID", coreBytes), nil
}

func (tx *Tx) FillTxID() (hash.Hash256, error) {
	id, err := tx.ComputeTxID()
	if err != nil {
		return hash.Hash256{}, err
	}
	tx.TxID = id
	return id, nil
}
