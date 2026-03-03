package tx

import (
	"CrowdShardChain/internal/hash"
)

type Tx struct {
	Version     uint32   `ccj:"version"`
	ChainID     string   `ccj:"chainId"`
	ShardID     uint32   `ccj:"shardId"`
	Time        uint64   `ccj:"time"`
	From        [20]byte `ccj:"from"`
	Nonce       uint64   `ccj:"nonce"`
	Fee         uint64   `ccj:"fee"`
	PayloadType string   `ccj:"payloadType"`
	Payload     []byte   `ccj:"payload"`

	TxID   hash.Hash256 `ccj:"txId"`
	PubKey []byte       `ccj:"pubKey"`
	Sig    []byte       `ccj:"sig"`
}

type TxCore struct {
	Version     uint32 `ccj:"version"`
	ChainID     string `ccj:"chainId"`
	ShardID     uint32 `ccj:"shardId"`
	Time        uint64 `ccj:"time"`
	From        string `ccj:"from"`
	Nonce       uint64 `ccj:"nonce"`
	Fee         uint64 `ccj:"fee"`
	PayloadType string `ccj:"payloadType"`
	Payload     string `ccj:"payload"`
}

func (tx *Tx) Core() TxCore {
	return TxCore{
		Version:     tx.Version,
		ChainID:     tx.ChainID,
		ShardID:     tx.ShardID,
		Time:        tx.Time,
		From:        hash.BytesToHex(tx.From[:]),
		Nonce:       tx.Nonce,
		Fee:         tx.Fee,
		PayloadType: tx.PayloadType,
		Payload:     hash.BytesToHex(tx.Payload),
	}
}
