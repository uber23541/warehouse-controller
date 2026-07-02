package domain

type Product struct {
	ID           int64
	ProductName  string
	Manufacturer string
	Category     string
	Price        int64
	Count        int32
}

type SearchProductsParams struct {
	ProductName  *string
	Manufacturer *string
	Category     *string
	MinPrice     *int64
	MaxPrice     *int64
	Limit        int32
	Offset       int32
}

type PatchProductParams struct {
	ID           int64
	ProductName  *string
	Manufacturer *string
	Category     *string
	Price        *int64
	Count        *int32
}

type DeleteProductParams struct {
	ID int64
}

type RestoreProductParams struct {
	ID int64
}

type GetProductParams struct {
	ID int64
}

type CreateProductParams struct {
	ProductName  string
	Manufacturer string
	Category     string
	Price        int64
	Count        int32
}

type ListProductsParams struct {
	Limit  int32
	Offset int32
}
