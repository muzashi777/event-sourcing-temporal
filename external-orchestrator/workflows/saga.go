package workflows

import (
	"time"

	"external-orchestrator/core"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ชื่อ Activity ที่เราจะเรียก (ต้องตรงกับที่ Inventory/Payment Service Register ไว้)
const (
	ActivityReserveStock   = "ReserveStock"
	ActivityReleaseStock   = "ReleaseStock"
	ActivityProcessPayment = "ProcessPayment"
)

// ชื่อ Task Queue ของแต่ละ Service
const (
	QueueInventory = "inventory-queue"
	QueuePayment   = "payment-queue"
)

func OrderSagaWorkflow(ctx workflow.Context, req core.CreateOrderRequest) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Order Saga started", "OrderID", req.OrderID)

	// --- Config: ตั้งค่า Retry Policy พื้นฐาน ---
	retryPolicy := temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    100 * time.Second,
		MaximumAttempts:    3, // ลอง 3 ครั้ง
	}

	// -----------------------------------------------------
	// STEP 1: Reserve Stock (เรียก Inventory Service)
	// -----------------------------------------------------
	inventoryOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		TaskQueue:           QueueInventory, // ส่งไปหา Inventory Worker
		RetryPolicy:         &retryPolicy,
	}
	ctx1 := workflow.WithActivityOptions(ctx, inventoryOptions)

	err := workflow.ExecuteActivity(ctx1, ActivityReserveStock, req.ProductID, req.Qty).Get(ctx1, nil)
	if err != nil {
		// ถ้าจองของไม่ได้ (เช่น Hard Check ไม่ผ่าน) -> ไม่ต้องทำอะไรต่อ
		logger.Error("Failed to reserve stock", "Error", err)
		return err
	}

	// -----------------------------------------------------
	// STEP 2: Process Payment (เรียก Payment Service)
	// -----------------------------------------------------
	paymentOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		TaskQueue:           QueuePayment, // ส่งไปหา Payment Worker
		RetryPolicy:         &retryPolicy,
	}
	ctx2 := workflow.WithActivityOptions(ctx, paymentOptions)

	err = workflow.ExecuteActivity(ctx2, ActivityProcessPayment, req.OrderID, req.Amount).Get(ctx2, nil)
	if err != nil {
		// !!! เกิดปัญหาตอนจ่ายเงิน !!!
		logger.Error("Payment failed. Starting compensation...", "Error", err)

		// -----------------------------------------------------
		// COMPENSATE: Release Stock (คืนของ)
		// -----------------------------------------------------
		// เราต้องใช้ Context เดิม (ctx1) หรือสร้างใหม่ที่ส่งไป Inventory Queue
		// แต่ห้ามใช้ ctx ที่มี parent cancel (ต้องใช้ DisconnectedContext ถ้า Workflow โดนยกเลิก)

		compensateCtx, _ := workflow.NewDisconnectedContext(ctx) // เพื่อให้ทำงานต่อได้แม้ Workflow หลักจะ Error
		compensateOpts := workflow.WithActivityOptions(compensateCtx, inventoryOptions)

		errCompensate := workflow.ExecuteActivity(compensateOpts, ActivityReleaseStock, req.ProductID, req.Qty).Get(compensateCtx, nil)

		if errCompensate != nil {
			logger.Error("Failed to compensate stock!", "Error", errCompensate)
			// ตรงนี้คือจุดวิกฤต (System Administrator ต้องเข้ามาดู Manual)
		}

		return err // ส่ง Error เดิมกลับไปบอกว่า Order Failed
	}

	logger.Info("Order Saga completed successfully")
	return nil
}
