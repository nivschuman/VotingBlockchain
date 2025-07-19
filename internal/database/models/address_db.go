package db_models

import "time"

type AddressDB struct {
	Ip         string     `gorm:"primaryKey;column:ip"`             // IP of address (primary key)
	Port       uint16     `gorm:"primaryKey;column:port"`           // Port number of address (primary key)
	NodeType   uint32     `gorm:"column:node_type;not null"`        // Type of node (e.g., full node)
	CreatedAt  *time.Time `gorm:"column:created_at;autoCreateTime"` // Timestamp when the peer was first recorded
	LastSeen   *time.Time `gorm:"column:last_seen"`                 // Timestamp of the last successful interaction with the address
	LastFailed *time.Time `gorm:"column:last_failed"`               // Timestamp of the last failed attempt to interact with address
}

func (AddressDB) TableName() string {
	return "addresses"
}
