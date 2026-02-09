package ports

import (
	"context"
	"external-orchestrator/core"

	"go.temporal.io/sdk/client"
)

// 1. ต้องการคนช่วยอ่านสต็อก (จาก Read Model)
type ProductRepository interface {
	GetProductView(ctx context.Context, productID string) (*core.ProductView, error)
}

// 2. ต้องการคนช่วยสั่ง Workflow (Temporal)
// หมายเหตุ: ใน Go เราใช้ client.Client ของ Temporal ได้เลย หรือจะห่อ Interface อีกชั้นก็ได้
// ในที่นี้เพื่อความง่าย เราจะใช้ client.Client ใน Handler โดยตรงครับ
type TemporalClient client.Client
