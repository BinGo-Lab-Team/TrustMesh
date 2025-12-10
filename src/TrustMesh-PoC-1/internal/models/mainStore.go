package models

// MainStore 主状态结构体
// 仅在创建时初始化，其余禁止更改顶层变量
type MainStore struct {
	ConnectionTable *ConnectionTable
	ProposalSate    *ProposalStore
}

// Init 初始化
func (m *MainStore) Init() {
	*m = MainStore{
		ConnectionTable: makeConnectionTable(),
		ProposalSate:    makeProposalStore(),
	}
}
