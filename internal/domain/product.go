package domain

type Product struct {
	ID           int64  `json:"id"`
	ProductName  string `json:"product_name"`
	Manufacturer string `json:"manufacturer"`
	Category     string `json:"category"`
	Price        int64  `json:"price"`
	Count        int32  `json:"count"`
}

type SearchProductsRequest struct {
	ProductName  *string `json:"product_name,omitempty"`
	Manufacturer *string `json:"manufacturer,omitempty"`
	Category     *string `json:"category,omitempty"`
	MinPrice     *int64  `json:"min_price,omitempty"`
	MaxPrice     *int64  `json:"max_price,omitempty"`
	Limit        *int32  `json:"limit,omitempty"`
	Offset       *int32  `json:"offset,omitempty"`
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
	ID           int64   `json:"id"`
	ProductName  *string `json:"product_name,omitempty"`
	Manufacturer *string `json:"manufacturer,omitempty"`
	Category     *string `json:"category,omitempty"`
	Price        *int64  `json:"price,omitempty"`
	Count        *int32  `json:"count,omitempty"`
}

type DeleteProductParams struct {
	ID int64 `json:"id"`
}

type RestoreProductParams struct {
	ID int64 `json:"id"`
}

type GetProductParams struct {
	ID int64 `json:"id"`
}

type CreateProductParams struct {
	ProductName  string `json:"product_name"`
	Manufacturer string `json:"manufacturer"`
	Category     string `json:"category"`
	Price        int64  `json:"price"`
	Count        int32  `json:"count"`
}

type ListProductsParams struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}
