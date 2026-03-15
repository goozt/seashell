package model

import "time"

// ValueRecord is a snapshot of an authority's token price at a given block height.
// Price is calculated using the velocity-of-money model:
//
//	velocity = tx_volume_last_50_blocks / circulating_supply
//	price    = base_price × (1 + sensitivity_k × velocity)
type ValueRecord struct {
	AuthorityID       string    `json:"authority_id"`
	BlockHeight       uint64    `json:"block_height"`
	Price             float64   `json:"price"`
	TransactionVolume int       `json:"transaction_volume"`
	CirculatingSupply int       `json:"circulating_supply"`
	Velocity          float64   `json:"velocity"`
	RecordedAt        time.Time `json:"recorded_at"`
}
