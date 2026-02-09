package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	mongoAdapter "payment-service/adapters/mongo"
	temporalAdapter "payment-service/adapters/temporal"
)

func main() {

	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017/?directConnection=true")
	temporalHost := getEnv("TEMPORAL_HOST", "127.0.0.1:7233")
	fmt.Printf("🔧 Config: Mongo=%s | Temporal=%s\n", mongoURI, temporalHost)

	// 1. Connect MongoDB
	mongoOpts := options.Client().ApplyURI(mongoURI)
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
		log.Fatal("Unable to create temporal client", err)
	}
	defer temporalClient.Close()

	// 3. Setup Adapters
	repo := mongoAdapter.NewMongoRepository(db)
	activities := temporalAdapter.NewPaymentActivities(repo)

	// 4. Start Worker
	// สังเกต: TaskQueue ชื่อ "payment-queue" (ต้องตรงกับที่ Orchestrator เรียก)
	w := worker.New(temporalClient, "payment-queue", worker.Options{})

	w.RegisterActivity(activities.ProcessPayment)

	log.Println("Payment Worker Started...")
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatal("Unable to start worker", err)
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
