package ports

import (
	"context"
	"inventory-service/core"
)

type InventoryRepository interface {
	// ดึง Event ทั้งหมดมาเพื่อ Replay
	GetEvents(ctx context.Context, productID string) ([]core.StockEvent, error)
	// บันทึก Event ใหม่ลง DB
	AppendEvent(ctx context.Context, event core.StockEvent) error
}
