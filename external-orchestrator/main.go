package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	httpAdapter "external-orchestrator/adapters/http"
	mongoAdapter "external-orchestrator/adapters/mongo"
	"external-orchestrator/workflows"
)

func main() {

	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017/?directConnection=true")
	temporalHost := getEnv("TEMPORAL_HOST", "127.0.0.1:7233")
	fmt.Printf("🔧 Config: Mongo=%s | Temporal=%s\n", mongoURI, temporalHost)

	// 1. Connect MongoDB
	mongoOpts := options.Client().ApplyURI(mongoURI) // หรือใช้ Env Var
	dbClient, err := mongo.Connect(context.Background(), mongoOpts)
	if err != nil {
		log.Fatal(err)
	}
	db := dbClient.Database("shop_db")

	// 2. Connect Temporal
	temporalClient, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatal("Unable to create client", err)
	}
	defer temporalClient.Close()

	// 3. Wiring Adapters (Dependency Injection)
	repo := mongoAdapter.NewMongoProductRepository(db)
	handler := httpAdapter.NewOrderHandler(repo, temporalClient)

	// --- ส่วนที่เพิ่ม: Start Workflow Worker ---
	// เพื่อให้ Temporal Server รู้ว่า Workflow "OrderSagaWorkflow" อยู่ที่นี่
	w := worker.New(temporalClient, "order-queue", worker.Options{})

	w.RegisterWorkflow(workflows.OrderSagaWorkflow) // ลงทะเบียนฟังก์ชัน

	// รัน Worker ใน Background (Goroutine)
	go func() {
		if err := w.Run(worker.InterruptCh()); err != nil {
			log.Fatalln("Unable to start workflow worker", err)
		}
	}()

	// 4. Start HTTP Server
	r := gin.Default()
	r.POST("/orders", handler.CreateOrder)

	log.Println("Orchestrator Service running on :8080")
	r.Run(":8080")
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
