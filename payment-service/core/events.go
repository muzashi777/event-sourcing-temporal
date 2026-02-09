package core

import "time"

const (
	EventPaymentProcessed = "PaymentProcessed"
	EventPaymentFailed    = "PaymentFailed"
)

type PaymentEvent struct {
	ID        string    `bson:"_id,omitempty"`
	OrderID   string    `bson:"order_id"` // ใช้ OrderID เป็น Stream ID
	Amount    int       `bson:"amount"`
	Type      string    `bson:"type"`
	Status    string    `bson:"status"` // SUCCESS / FAILED
	Timestamp time.Time `bson:"timestamp"`
}
