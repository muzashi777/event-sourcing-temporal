package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"

	"external-orchestrator/core"
	"external-orchestrator/ports"
)

type OrderHandler struct {
	Repo           ports.ProductRepository
	TemporalClient client.Client
}

func NewOrderHandler(repo ports.ProductRepository, tClient client.Client) *OrderHandler {
	return &OrderHandler{Repo: repo, TemporalClient: tClient}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req core.CreateOrderRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Body"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// --- 1. SOFT CHECK (Read Model) ---
	product, err := h.Repo.GetProductView(ctx, req.ProductID)
	fmt.Println("pruduct:", product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Check stock failed"})
		return
	}

	// ถ้าของใน Read Model หมด -> Fail Fast
	if product.AvailableStock < req.Qty {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Out of stock (Soft check)",
			"current_stock": product.AvailableStock,
		})
		return
	}

	// --- 2. START TEMPORAL WORKFLOW ---
	workflowOptions := client.StartWorkflowOptions{
		ID:        "order-" + req.OrderID, // Business ID
		TaskQueue: "order-queue",          // ชื่อคิวที่ Worker จะมารับงาน
	}

	// สั่งรัน Workflow ชื่อ "OrderSagaWorkflow"
	// ส่งข้อมูล req เข้าไปประมวลผลต่อ
	we, err := h.TemporalClient.ExecuteWorkflow(context.Background(), workflowOptions, "OrderSagaWorkflow", req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start workflow"})
		return
	}

	// ตอบกลับ 202 Accepted (รับเรื่องแล้ว)
	c.JSON(http.StatusAccepted, gin.H{
		"message":     "Order processing started",
		"workflow_id": we.GetID(),
		"run_id":      we.GetRunID(),
	})
}
