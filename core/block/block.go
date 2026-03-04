package block

import (
	"CrowdShardChain/core/tx"
	"CrowdShardChain/internal/ccj"
	"CrowdShardChain/internal/crypto"
	"CrowdShardChain/internal/hash"
	"CrowdShardChain/internal/merkle"
	"errors"
)

type Block struct {
	Header Header  `ccj:"header"`
	Txs    []tx.Tx `ccj:"txs"`
}

type Header struct {
	Version uint32 `ccj:"version"`
	ChainID string `ccj:"chainId"`

	Height uint64 `ccj:"height"`
	Time   uint64 `ccj:"time"`

	PrevHash hash.Hash256 `ccj:"prevHash"`

	TxRoot         hash.Hash256 `ccj:"txRoot"`
	ShardRootsRoot hash.Hash256 `ccj:"shardRootsRoot"`
	OutboxRoot     hash.Hash256 `ccj:"outboxRoot"`

	// frame fields（不入 BlockID）
	BlockID  hash.Hash256 `ccj:"blockId"`
	Proposer [20]byte     `ccj:"proposer"`
	PubKey   string       `ccj:"pubKey"` // hex string（CCJ 不支持 []byte）
	Sig      string       `ccj:"sig"`    // hex string
}

type HeaderCore struct {
	Version uint32 `ccj:"version"`
	ChainID string `ccj:"chainId"`

	Height uint64 `ccj:"height"`
	Time   uint64 `ccj:"time"`

	PrevHash string `ccj:"prevHash"`

	TxRoot         string `ccj:"txRoot"`
	ShardRootsRoot string `ccj:"shardRootsRoot"`
	OutboxRoot     string `ccj:"outboxRoot"`

	Proposer string `ccj:"proposer"` // hex(20 bytes)
}

func (h *Header) Core() HeaderCore {
	return HeaderCore{
		Version:        h.Version,
		ChainID:        h.ChainID,
		Height:         h.Height,
		Time:           h.Time,
		PrevHash:       h.PrevHash.Hex(),
		TxRoot:         h.TxRoot.Hex(),
		ShardRootsRoot: h.ShardRootsRoot.Hex(),
		OutboxRoot:     h.OutboxRoot.Hex(),
		Proposer:       hash.BytesToHex(h.Proposer[:]),
	}
}

// ------------------------- TxRoot -------------------------

// ComputeTxRoot：按区块内 tx 顺序计算 root（顺序有语义，不排序）
// 叶子：DomainHash("TXLEAF", txid[:])
// 空集合：DomainHash("TXROOT_EMPTY", {0x00})
func (b *Block) ComputeTxRoot() (hash.Hash256, error) {
	if b == nil {
		return hash.Hash256{}, errors.New("block: nil")
	}
	if len(b.Txs) == 0 {
		return hash.DomainHash("TXROOT_EMPTY", []byte{0x00}), nil
	}

	leaves := make([]hash.Hash256, 0, len(b.Txs))
	for i := range b.Txs {
		leaf := hash.DomainHash("TXLEAF", b.Txs[i].TxID.Bytes())
		leaves = append(leaves, leaf)
	}
	return merkle.Root(leaves), nil
}

func (b *Block) FillTxRoot() (hash.Hash256, error) {
	r, err := b.ComputeTxRoot()
	if err != nil {
		return hash.Hash256{}, err
	}
	b.Header.TxRoot = r
	return r, nil
}

// ------------------------- BlockID -------------------------

// ComputeBlockID：BlockID = DomainHash("BLKID", CCJ.Marshal(HeaderCore))
func (b *Block) ComputeBlockID() (hash.Hash256, error) {
	if b == nil {
		return hash.Hash256{}, errors.New("block: nil")
	}
	coreBytes, err := ccj.Marshal(b.Header.Core())
	if err != nil {
		return hash.Hash256{}, err
	}
	return hash.DomainHash("BLKID", coreBytes), nil
}

func (b *Block) FillBlockID() (hash.Hash256, error) {
	id, err := b.ComputeBlockID()
	if err != nil {
		return hash.Hash256{}, err
	}
	b.Header.BlockID = id
	return id, nil
}

// SigMsg = DomainHash("BLKSIG", blockid[:])
func sigMsg(blockID hash.Hash256) hash.Hash256 {
	return hash.DomainHash("BLKSIG", blockID.Bytes())
}

// ------------------------- Sign / Verify -------------------------

// Sign：提议者签名（只签 BlockID）
// - 会先 FillTxRoot（确保 header.txRoot 正确）
// - 会再 FillBlockID
// - PubKey/Sig 用 hex string 存储（适配 CCJ）
func (b *Block) Sign(privKey []byte) error {
	if b == nil {
		return errors.New("block: nil")
	}

	if _, err := b.FillTxRoot(); err != nil {
		return err
	}
	id, err := b.FillBlockID()
	if err != nil {
		return err
	}

	pub, err := crypto.PubKey(privKey) // 返回 []byte(32)
	if err != nil {
		return err
	}
	b.Header.PubKey = hash.BytesToHex(pub)

	msg := sigMsg(id)
	sig, err := crypto.Sign(privKey, msg.Bytes()) // 返回 []byte(64)
	if err != nil {
		return err
	}
	b.Header.Sig = hash.BytesToHex(sig)
	return nil
}

// Verify：
// 1) 校验 TxRoot 是否匹配 Txs
// 2) 重算 BlockID 是否匹配 header.blockId
// 3) 验签：Verify(pubkey, DomainHash("BLKSIG", blockid[:]), sig)
// 4) 强烈建议：校验 proposer == Addr(pubkey)（按你地址规则实现）
func (b *Block) Verify() error {
	if b == nil {
		return errors.New("block: nil")
	}

	// 1) TxRoot
	txr, err := b.ComputeTxRoot()
	if err != nil {
		return err
	}
	if b.Header.TxRoot != txr {
		return errors.New("block: txRoot mismatch")
	}

	// 2) BlockID
	id, err := b.ComputeBlockID()
	if err != nil {
		return err
	}
	if b.Header.BlockID != id {
		return errors.New("block: blockId mismatch")
	}

	// 3) signature
	pubBytes, err := hash.HexToBytes(b.Header.PubKey)
	if err != nil {
		return errors.New("block: invalid pubKey hex")
	}
	sigBytes, err := hash.HexToBytes(b.Header.Sig)
	if err != nil {
		return errors.New("block: invalid sig hex")
	}

	msg := sigMsg(id).Bytes()
	ok, err := crypto.Verify(pubBytes, msg, sigBytes)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("block: signature verify failed")
	}

	// 4) proposer binding（按你的地址规则补上）
	// addr := hash.Hash160(pubBytes)
	// if addr != b.Header.Proposer { return errors.New("block: proposer != addr(pubKey)") }

	return nil
}
