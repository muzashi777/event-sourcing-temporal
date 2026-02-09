// scripts/init-mongo.js

try {
    rs.status();
} catch (err) {
    try {
        rs.initiate({_id: 'rs0', members: [{_id: 0, host: 'localhost:27017'}]});
        print("✅ Replica Set Initiated via Script");
        sleep(5000); // รอให้เป็น Primary
    } catch (e) {
        print("ℹ️ Replica Set already active or handled by healthcheck");
    }
}
// 1. เลือก Database
db = db.getSiblingDB('shop_db');

print("🚀 Starting Database Initialization for 'shop_db'...");

// ==========================================
// A. Collection: events (Event Store)
// ==========================================
// สร้าง Collection (ถ้ายังไม่มี)
db.createCollection("events");

// 🔥 สร้าง Index: ห้าม Version ซ้ำในสินค้าตัวเดิม (Optimistic Locking)
db.events.createIndex({ "stream_id": 1, "version": 1 }, { unique: true });
print("✅ Index created: events (stream_id + version)");

// 📝 Mock Data: เติมสต็อกเริ่มต้น (Seed Data)
// เราจะจำลองว่ามีการเติมของเข้ามาแล้ว (StockAdded)
db.events.insertMany([
  {
    stream_id: "iphone-15",
    type: "StockAdded",
    qty: 100,
    version: 1,
    timestamp: new Date()
  },
  {
    stream_id: "macbook-pro",
    type: "StockAdded",
    qty: 50,
    version: 1,
    timestamp: new Date()
  }
]);
print("✅ Mock Data inserted: events (StockAdded)");

// ==========================================
// B. Collection: products_view (Read Model)
// ==========================================
db.createCollection("products_view");

// 🔥 สร้าง Index: ห้าม Product ID ซ้ำ (Idempotency / Fast Read)
db.products_view.createIndex({ "product_id": 1 }, { unique: true });
print("✅ Index created: products_view (product_id)");

// 📝 Mock Data: เติมข้อมูลให้ตรงกับ Event
// (เพื่อให้ API อ่านได้เลยโดยไม่ต้องรอ Projector รันครั้งแรก)
db.products_view.insertMany([
  {
    product_id: "iphone-15",
    available_stock: 100,
    last_version: 1
  },
  {
    product_id: "macbook-pro",
    available_stock: 50,
    last_version: 1
  }
]);
print("✅ Mock Data inserted: products_view");

// ==========================================
// C. Collection: checkpoints (Projector State)
// ==========================================
db.createCollection("checkpoints");
print("✅ Collection created: checkpoints");

print("🎉 Database Initialization Completed!");