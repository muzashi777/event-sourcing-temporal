package temporal

import (
	"context"
	"errors"
	"time"

	"payment-service/core"
	"payment-service/ports"
)

type PaymentActivities struct {
	Repo ports.PaymentRepository
}

func NewPaymentActivities(repo ports.PaymentRepository) *PaymentActivities {
	return &PaymentActivities{Repo: repo}
}

// Activity: ProcessPayment
func (a *PaymentActivities) ProcessPayment(ctx context.Context, orderID string, amount int) error {
	// --- จำลอง Logic การตัดเงิน ---
	// ในของจริงอาจจะยิง API ไปหา Stripe / Omise / Bank

	if amount > 10000 {
		return errors.New("insufficient funds (simulated)")
	}

	// ถ้าตัดเงินผ่าน ให้บันทึก Event
	event := core.PaymentEvent{
		OrderID:   orderID,
		Amount:    amount,
		Type:      core.EventPaymentProcessed,
		Status:    "SUCCESS",
		Timestamp: time.Now(),
	}

	err := a.Repo.AppendEvent(ctx, event)
	if err != nil {
		return err
	}

	return nil // Return nil แปลว่า Activity สำเร็จ Temporal จะไปต่อ
}
