package mongo

import (
	"context"
	"inventory-service/core"
	"inventory-service/ports"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepository struct {
	Collection *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) ports.InventoryRepository {
	return &MongoRepository{
		Collection: db.Collection("events"), // เก็บลง collection นี้
	}
}

func (r *MongoRepository) GetEvents(ctx context.Context, productID string) ([]core.StockEvent, error) {
	filter := bson.M{"stream_id": productID}
	// สำคัญมาก: ต้อง Sort ตามเวลาหรือ Version เพื่อให้ Replay ถูกลำดับ
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})

	cursor, err := r.Collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	var events []core.StockEvent
	if err = cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *MongoRepository) AppendEvent(ctx context.Context, event core.StockEvent) error {
	_, err := r.Collection.InsertOne(ctx, event)
	return err
}
