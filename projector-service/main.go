package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ---------------------------------------------------------
// 1. Data Structures
// ---------------------------------------------------------

// Checkpoint เอาไว้เก็บว่าอ่านถึงไหนแล้ว (Resume Token)
type Checkpoint struct {
	ID          string      `bson:"_id"`          // ID ของ Projector (เผื่อมีหลายตัว)
	ResumeToken interface{} `bson:"resume_token"` // Token ของ MongoDB Change Stream
}

// StockEvent หน้าตาของ Event ที่เราจะอ่านจาก Stream
type StockEvent struct {
	StreamID  string    `bson:"stream_id"` // Product ID
	Type      string    `bson:"type"`
	Qty       int       `bson:"qty"`
	Version   int       `bson:"version"` // เก็บไว้ดูเล่น (ไม่ได้ใช้คำนวณใน view)
	Timestamp time.Time `bson:"timestamp"`
}

// ---------------------------------------------------------
// 2. Main Logic
// ---------------------------------------------------------

func main() {
	// A. Connect MongoDB
	// หมายเหตุ: ต้องใช้ directConnection=true เสมอ เมื่อต่อ Replica Set จากเครื่อง Local

	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017/?directConnection=true")

	fmt.Printf("🔧 Config: Mongo=%s \n", mongoURI)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("❌ Connection Failed:", err)
	}
	defer client.Disconnect(context.Background())

	// B. Setup Collections
	db := client.Database("shop_db")
	eventsCol := db.Collection("events")          // Source (Event Store)
	viewCol := db.Collection("products_view")     // Target (Read Model)
	checkpointCol := db.Collection("checkpoints") // State (Resume Token)

	fmt.Println("🚀 Projector Service Starting...")

	// C. Load Resume Token (กู้คืนจุดล่าสุดที่อ่านค้างไว้)
	var resumeToken interface{}
	var checkpoint Checkpoint

	// พยายามหา Checkpoint ชื่อ "main_projector"
	err = checkpointCol.FindOne(context.Background(), bson.M{"_id": "main_projector"}).Decode(&checkpoint)
	if err == nil {
		resumeToken = checkpoint.ResumeToken
		fmt.Println("🔄 Resumed from last checkpoint.")
	} else {
		fmt.Println("🆕 No checkpoint found. Starting from now (or beginning).")
	}

	// D. Configure Change Stream
	streamOpts := options.ChangeStream()
	if resumeToken != nil {
		// กรณี 1: มี Token (เคยรันแล้ว) -> ทำต่อจากเดิม
		fmt.Println("🔄 Resuming from Checkpoint...")
		streamOpts.SetResumeAfter(resumeToken)
	} else {
		// กรณี 2: ไม่มี Token (เพิ่งลบ Checkpoint หรือรันครั้งแรก)
		// 💥 ต้องสั่งให้เริ่มอ่านตั้งแต่ "จุดเริ่มต้นของเวลา" (Timestamp 1, 0)
		fmt.Println("🆕 No Checkpoint. Replaying ALL history from beginning...")

		// ต้อง import "go.mongodb.org/mongo-driver/bson/primitive" ข้างบนด้วย
		startOfTime := primitive.Timestamp{T: 1, I: 0}
		streamOpts.SetStartAtOperationTime(&startOfTime)
	}
	// ถ้าไม่มี Token โดย default มันจะเริ่มอ่าน Event ใหม่ที่เกิดขึ้นหลังจากนี้ (Real-time)
	// แต่ถ้าอยากให้อ่านย้อนหลังทั้งหมดตั้งแต่เริ่มโลก ให้ใช้:
	// streamOpts.SetStartAtOperationTime(&primitive.Timestamp{T: 1, I: 0})

	// Filter: สนใจแค่การ Insert ข้อมูลใหม่ลง Event Store
	pipeline := mongo.Pipeline{
		// {{"$match", bson.D{{"operationType", "insert"}}}},
	}

	stream, err := eventsCol.Watch(context.Background(), pipeline, streamOpts)
	if err != nil {
		log.Fatal("❌ Error starting stream:", err)
	}
	defer stream.Close(context.Background())

	fmt.Println("👀 Watching for events...")

	// E. Infinite Loop (Processing)
	for stream.Next(context.Background()) {
		// โครงสร้างข้อมูลที่ Change Stream ส่งมา
		var changeEvent struct {
			ID           interface{} `bson:"_id"`          // นี่คือ Resume Token ของ Event นี้
			FullDocument StockEvent  `bson:"fullDocument"` // ข้อมูล Event จริงๆ
		}

		if err := stream.Decode(&changeEvent); err != nil {
			log.Println("⚠️ Error decoding event:", err)
			continue
		}

		event := changeEvent.FullDocument
		token := changeEvent.ID

		// 1. Process Logic (อัปเดต Read Model)
		err := processEvent(viewCol, event)
		if err != nil {
			log.Printf("❌ Failed to process event %s: %v\n", event.StreamID, err)
			// ใน Production: อาจจะ Retry หรือส่งเข้า Dead Letter Queue
			// แต่ Projector ไม่ควรหยุดทำงาน
		}

		// 2. Save Checkpoint (บันทึกว่าทำถึงไหนแล้ว)
		// บันทึก Token ลง DB เพื่อที่ถ้าโปรแกรมดับ เปิดมาใหม่จะได้ทำต่อจากตรงนี้
		_, err = checkpointCol.UpdateOne(context.Background(),
			bson.M{"_id": "main_projector"},
			bson.M{"$set": bson.M{"resume_token": token}},
			options.Update().SetUpsert(true), // ถ้าไม่มีให้สร้างใหม่
		)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to save checkpoint: %v\n", err)
		}
	}
	if err := stream.Err(); err != nil {
		log.Fatal("❌ Stream Error: ", err) // <--- มันฟ้องว่าอะไรครับ?
	}
	fmt.Println("👋 Stream closed gracefully (Invalidate?).")
}

// ---------------------------------------------------------
// 3. Business Logic (Safe Idempotent Version)
// ---------------------------------------------------------

func processEvent(viewCol *mongo.Collection, event StockEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. คำนวณยอดที่จะเปลี่ยนแปลง (Change)
	change := 0
	switch event.Type {
	case "StockReserved":
		change = -event.Qty // จองของ = ลบ
	case "StockReleased", "StockAdded":
		change = event.Qty // คืนของ/เติมของ = บวก
	default:
		return nil // Event ที่ไม่รู้จัก ข้ามไป
	}

	fmt.Printf("⚡ Processing Event: %s (v.%d) | Change: %d | Product: %s\n",
		event.Type, event.Version, change, event.StreamID)

	// 2. เช็คข้อมูลปัจจุบันใน DB ก่อน (Check Phase)
	filter := bson.M{"product_id": event.StreamID}

	var currentDoc struct {
		LastVersion int `bson:"last_version"`
	}

	// พยายามหาเอกสารเก่า
	err := viewCol.FindOne(ctx, filter).Decode(&currentDoc)

	if err == nil {
		// ---------------------------------------------------
		// กรณี A: เจอข้อมูลเดิม (Found)
		// ---------------------------------------------------

		// กฎเหล็ก: ห้ามถอยหลังลงคลอง
		// ถ้า Version ใน DB ใหม่กว่าหรือเท่ากับ Event ที่กำลังเข้ามา แปลว่าเราเคย Process ไปแล้ว
		if currentDoc.LastVersion >= event.Version {
			log.Printf("   ⚠️ Skipped: Event v.%d is older/equal to DB v.%d\n", event.Version, currentDoc.LastVersion)
			return nil // จบการทำงานแบบปกติ (ถือว่าสำเร็จ)
		}

		// ถ้า Event ใหม่กว่า -> อัปเดตยอด
		update := bson.M{
			"$inc": bson.M{"available_stock": change},
			"$set": bson.M{"last_version": event.Version},
		}

		_, err := viewCol.UpdateOne(ctx, filter, update)
		if err != nil {
			return fmt.Errorf("failed to update view: %w", err)
		}

	} else if err == mongo.ErrNoDocuments {
		// ---------------------------------------------------
		// กรณี B: ไม่เจอข้อมูลเดิม (Not Found) -> สินค้าใหม่
		// ---------------------------------------------------

		newDoc := bson.M{
			"product_id":      event.StreamID,
			"available_stock": change, // ยอดตั้งต้นเท่ากับค่า change เลย
			"last_version":    event.Version,
			// คุณอาจเพิ่ม field อื่นๆ เช่น updated_at ตรงนี้
		}

		_, err := viewCol.InsertOne(ctx, newDoc)
		if err != nil {
			// กันเหนียว: กรณีมี Race Condition (Projector 2 ตัวแย่งกันสร้าง)
			if mongo.IsDuplicateKeyError(err) {
				log.Println("   ⚠️ Insert skipped (Duplicate Key). Another worker processed it.")
				return nil
			}
			return fmt.Errorf("failed to insert view: %w", err)
		}

	} else {
		// ---------------------------------------------------
		// กรณี C: Error อื่นๆ (เช่น DB Connection หลุด)
		// ---------------------------------------------------
		return fmt.Errorf("error finding document: %w", err)
	}

	fmt.Println("   ✅ View Updated Successfully.")
	return nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
