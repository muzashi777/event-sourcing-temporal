package main

import (
	"context"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	mongoAdapter "inventory-service/adapters/mongo"
	temporalAdapter "inventory-service/adapters/temporal"
)

func main() {

	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017/?directConnection=true")
	temporalHost := getEnv("TEMPORAL_HOST", "127.0.0.1:7233")

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
	activities := temporalAdapter.NewInventoryActivities(repo)

	// 4. Start Worker
	// "order-queue" คือชื่อ Queue ที่เราตั้งไว้ใน Orchestrator
	// หรือจะแยกเป็น "inventory-queue" ก็ได้แล้วแต่ design
	w := worker.New(temporalClient, "inventory-queue", worker.Options{})

	// Register Functions ให้ Temporal รู้จัก
	w.RegisterActivity(activities.ReserveStock)
	w.RegisterActivity(activities.ReleaseStock)

	log.Println("Inventory Worker Started...")
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
