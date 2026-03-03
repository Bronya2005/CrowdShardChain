package state

import (
	"CrowdShardChain/internal/ccj"
	"CrowdShardChain/internal/hash"
)

func EncodeAccountSnap(s AccountSnap) ([]byte, error) {
	return ccj.Marshal(s)
}

func DecodeAccountSnap(b []byte) (AccountSnap, error) {
	var s AccountSnap
	if err := ccj.Unmarshal(b, &s); err != nil {
		return AccountSnap{}, err
	}
	return s, nil
}

func EncodeTaskSnap(s TaskSnap) ([]byte, error) {
	return ccj.Marshal(s)
}

func DecodeTaskSnap(b []byte) (TaskSnap, error) {
	var s TaskSnap
	if err := ccj.Unmarshal(b, &s); err != nil {
		return TaskSnap{}, err
	}
	return s, nil
}

func EncodeAccount(addr [20]byte, a *account) ([]byte, error) {
	snap := AccountSnap{
		AddrHex: hash.BytesToHex(addr[:]),
		Balance: a.Balance,
		Nonce:   a.Nonce,
	}
	return EncodeAccountSnap(snap)
}

func EncodeTask(taskId [32]byte, t *Task) ([]byte, error) {
	deps := make(map[string]uint64)
	if t != nil && t.Deposits != nil {
		deps = make(map[string]uint64, len(t.Deposits))
		for k, v := range t.Deposits {
			keyHex := hash.BytesToHex(k[:])
			deps[keyHex] = v
		}
	}

	snap := TaskSnap{
		TaskIDHex:   hash.BytesToHex(taskId[:]),
		Requester:   t.Requester,
		Title:       t.Title,
		Description: t.Description,
		Status:      uint8(t.Status),
		RewardPool:  t.RewardPool,
		Deposits:    deps,
		Winner:      t.Winner,
		Settled:     t.Settled,
	}
	return EncodeTaskSnap(snap)
}
