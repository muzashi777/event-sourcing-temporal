package mongo

import (
	"context"
	"payment-service/core"
	"payment-service/ports"

	"go.mongodb.org/mongo-driver/mongo"
)

type MongoRepository struct {
	Collection *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) ports.PaymentRepository {
	// แยก Database หรือ Collection ให้ชัดเจน
	return &MongoRepository{
		Collection: db.Collection("payment_events"),
	}
}

func (r *MongoRepository) AppendEvent(ctx context.Context, event core.PaymentEvent) error {
	_, err := r.Collection.InsertOne(ctx, event)
	return err
}
