package mongo

import (
	"context"
	"errors"
	"external-orchestrator/core"
	"external-orchestrator/ports"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoProductRepository struct {
	Collection *mongo.Collection
}

// ฟังก์ชันสร้างตัวช่วย (Constructor)
func NewMongoProductRepository(db *mongo.Database) ports.ProductRepository {
	return &MongoProductRepository{
		Collection: db.Collection("products_view"), // อ่านจาก Read Model
	}
}

func (r *MongoProductRepository) GetProductView(ctx context.Context, productID string) (*core.ProductView, error) {
	var product core.ProductView

	// Query หา ID สินค้า
	err := r.Collection.FindOne(ctx, bson.M{"product_id": productID}).Decode(&product)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	return &product, nil
}
