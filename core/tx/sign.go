package tx

import (
	"CrowdShardChain/internal/crypto"
	"CrowdShardChain/internal/hash"
	"errors"
)

func (tx *Tx) SigMsg(txid hash.Hash256) hash.Hash256 {
	return hash.DomainHash("TXSIG", txid.Bytes())
}
func (tx *Tx) Sign(numShards uint32, pubkey []byte, privKey []byte) ([]byte, error) {
	if err := tx.ForceShard(numShards); err != nil {
		return nil, err
	}
	id, err := tx.FillTxID()
	if err != nil {
		return nil, err
	}
	msg := tx.SigMsg(id)
	sig, err := crypto.Sign(privKey, msg.Bytes())
	tx.PubKey = pubkey
	tx.Sig = sig
	return sig, nil
}

func (tx *Tx) Verify(numShards uint32) (bool, error) {
	if err := tx.ForceShard(numShards); err != nil {
		return false, err
	}

	id, err := tx.ComputeTxID()
	if err != nil {
		return false, err
	}
	if tx.TxID != id {
		return false, errors.New("交易校验失败：交易ID不匹配")
	}

	msg := tx.SigMsg(id).Bytes()
	ok, err := crypto.Verify(tx.PubKey, msg, tx.Sig)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, errors.New("交易校验失败，签名不匹配")
	}
	return ok, nil
}
