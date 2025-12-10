package table

// Peer 节点表
type Peer struct {
	ID         uint64 `gorm:"primary_key"`
	NodeID     []byte `gorm:"uniqueIndex:idx_node_id;uniqueIndex:idx_node_id_addr;not null"`
	Address    string `gorm:"uniqueIndex:idx_node_addr;uniqueIndex:idx_node_id_addr;not null"`
	Reputation uint16 `gorm:"not null"`
	LastSeen   uint64 `gorm:"not null;index"`
	Status     string `gorm:"not null"`
}
