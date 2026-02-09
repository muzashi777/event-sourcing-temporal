package core

import "time"

// ชื่อ Event (Constants) เพื่อป้องกันการพิมพ์ผิด
const (
	EventStockReserved = "StockReserved"
	EventStockReleased = "StockReleased" // ใช้ตอน Compensate
	EventStockAdded    = "StockAdded"    // ใช้ตอนเติมของ
)

// โครงสร้าง Event ที่จะเก็บลง MongoDB
type StockEvent struct {
	ID        string    `bson:"_id,omitempty"` // Mongo generates this
	Version   int       `bson:"version"`
	StreamID  string    `bson:"stream_id"` // Product ID
	Type      string    `bson:"type"`
	Qty       int       `bson:"qty"`
	Timestamp time.Time `bson:"timestamp"`
}
