package state

type account struct {
	Nonce   uint64
	Balance uint64
}

func (s *ShardState) GetAccount(addr20 [20]byte) *account {
	acc, ok := s.accounts[addr20]
	if !ok {
		acc = &account{}
		s.accounts[addr20] = acc
	}
	return acc
}

// cloneAccount 深复制 account，确保修改副本不会影响原始 account
func cloneAccount(a *account) *account {
	if a == nil {
		return nil
	}
	cp := *a
	return &cp
}
