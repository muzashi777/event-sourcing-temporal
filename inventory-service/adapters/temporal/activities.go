package temporal

import (
	"context"
	"errors"
	"time"

	"inventory-service/core"
	"inventory-service/ports"
)

type InventoryActivities struct {
	Repo ports.InventoryRepository
}

func NewInventoryActivities(repo ports.InventoryRepository) *InventoryActivities {
	return &InventoryActivities{Repo: repo}
}

// Activity 1: จองสต็อก (Hard Check)
func (a *InventoryActivities) ReserveStock(ctx context.Context, productID string, qty int) error {
	// 1. Replay
	events, _ := a.Repo.GetEvents(ctx, productID)
	agg := core.NewInventoryAggregate(productID)
	agg.Replay(events) // Replay จะได้ agg.LastVersion ออกมาด้วย (สมมติเป็น 5)

	// 2. Validate Stock
	if agg.CurrentStock < qty {
		return errors.New("out of stock")
	}

	// 3. Prepare New Event
	newEvent := core.StockEvent{
		StreamID:  productID,
		Type:      core.EventStockReserved,
		Qty:       qty,
		Timestamp: time.Now(),
		Version:   agg.LastVersion + 1, // บังคับว่าเป็น Version 6 เท่านั้น
	}

	// 4. Append (ถ้ามีคนอื่นแย่งเขียน Version 6 ไปก่อนหน้านี้เสี้ยววินาที ตรงนี้จะ Error)
	err := a.Repo.AppendEvent(ctx, newEvent)
	if err != nil {
		// ถ้า MongoDB ฟ้องว่า Duplicate Key
		return errors.New("concurrency error: please retry")
	}

	return nil
}

// Activity 2: คืนสต็อก (Compensate)
// จะถูกเรียกเมื่อ Payment พัง
// adapters/temporal/activities.go

func (a *InventoryActivities) ReleaseStock(ctx context.Context, productID string, qty int) error {
	// 1. Load History (ต้องดึงประวัติเก่ามาก่อน เพื่อหา Version ล่าสุด)
	events, err := a.Repo.GetEvents(ctx, productID)
	if err != nil {
		return err
	}

	// 2. Replay (เพื่อเอาค่า LastVersion)
	agg := core.NewInventoryAggregate(productID)
	agg.Replay(events)

	// หมายเหตุ: ตอนคืนของ ปกติเราไม่ต้องเช็คว่า agg.CurrentStock พอไหม
	// เพราะการคืนของคือการบวกเพิ่ม ย่อมทำได้เสมอ

	// 3. Prepare New Event (สร้าง Event คืนของ โดยเพิ่ม Version ไป 1)
	newEvent := core.StockEvent{
		StreamID:  productID,
		Type:      core.EventStockReleased, // เป็น Type คืนของ
		Qty:       qty,
		Timestamp: time.Now(),
		Version:   agg.LastVersion + 1, // ✅ ต้องบวก 1 จากตัวล่าสุดเสมอ
	}

	// 4. Append ลง DB
	err = a.Repo.AppendEvent(ctx, newEvent)
	if err != nil {
		// ถ้าบังเอิญมีคนแย่งเขียน Version นี้ตัดหน้าไปพอดี (Concurrency)
		// Temporal จะจับ Error นี้แล้ว Retry ให้เองตาม Policy -> กลับไปทำข้อ 1 ใหม่ -> ได้ Version ใหม่
		return err
	}

	return nil
}
