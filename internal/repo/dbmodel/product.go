package dbmodel

type Product struct {
	ID           int64
	ProductName  string
	Manufacturer string
	Category     string
	Count        int32
	Price        int64
}

type ProductFilter struct {
	ProductName  *string
	Manufacturer *string
	Category     *string
	MinPrice     *int64
	MaxPrice     *int64
	Limit        int32
	Offset       int32
}

type ProductPatch struct {
	ProductName  *string
	Manufacturer *string
	Category     *string
	Count        *int32
	Price        *int64
}