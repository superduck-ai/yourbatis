package projectionconformance

type OrderProjection struct {
	Price      int64 `db:"price"`
	Discount   int64 `db:"discount"`
	FinalPrice int64 `db:"final_price"`
}
