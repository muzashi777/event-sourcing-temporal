package core

// สิ่งที่ลูกค้าส่งมา
type CreateOrderRequest struct {
	OrderID   string `json:"order_id"`
	ProductID string `json:"product_id"`
	Qty       int    `json:"qty"`
	Amount    int    `json:"amount"` // ยอดเงินที่ต้องตัด
}

// สิ่งที่เราอ่านจาก Read Model (MongoDB)
type ProductView struct {
	ProductID      string `bson:"product_id"`
	AvailableStock int    `bson:"available_stock"`
}
