package ports

import (
	"context"
	"payment-service/core"
)

type PaymentRepository interface {
	AppendEvent(ctx context.Context, event core.PaymentEvent) error
}
