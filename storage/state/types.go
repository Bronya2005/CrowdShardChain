package state

type AccountSnap struct {
	AddrHex string `json:"addrHex"`
	Balance uint64 `json:"balance"`
	Nonce   uint64 `json:"nonce"`
}

type TaskSnap struct {
	TaskIDHex string `json:"taskIdHex"`
	Requester string `json:"requester"`

	Title       string `json:"title"`
	Description string `json:"description"`

	Status     uint8  `json:"status"`
	RewardPool uint64 `json:"rewardPool"`

	Deposits map[string]uint64 `json:"deposits"` // workerAddrHex -> deposit

	Winner  string `json:"winner"`
	Settled bool   `json:"settled"`
}
