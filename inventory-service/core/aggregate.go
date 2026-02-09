package core

// InventoryAggregate คือตัวแทนของสินค้า 1 ชิ้นใน RAM
type InventoryAggregate struct {
	ProductID    string
	CurrentStock int
	LastVersion  int // ใช้เก็บ version ล่าสุดของ Event Sourcing
}

// สร้าง Aggregate เปล่าๆ
func NewInventoryAggregate(productID string) *InventoryAggregate {
	return &InventoryAggregate{
		ProductID:    productID,
		CurrentStock: 0,
	}
}

// ฟังก์ชัน Replay: รับ Event เข้ามา 1 ตัว แล้วอัปเดตสถานะตัวเอง
func (a *InventoryAggregate) Apply(event StockEvent) {
	switch event.Type {
	case EventStockAdded:
		a.CurrentStock += event.Qty
	case EventStockReserved:
		a.CurrentStock -= event.Qty
	case EventStockReleased:
		a.CurrentStock += event.Qty
	}
	a.LastVersion = event.Version
}

// Replay ทีเดียวหลายตัว
func (a *InventoryAggregate) Replay(events []StockEvent) {
	for _, evt := range events {
		a.Apply(evt)
	}
}
