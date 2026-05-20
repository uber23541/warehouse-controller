package domain

type Product struct {
	ID           int64  `json:"id"           example:"42"`
	ProductName  string `json:"product_name" example:"Молоток"`
	Manufacturer string `json:"manufacturer" example:"Зубр"`
	Category     string `json:"category"     example:"Инструменты"`
	Price        int64  `json:"price"        example:"1299"`
	Count        int32  `json:"count"        example:"50"`
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
	ProductName  *string `form:"product_name"`
	Manufacturer *string `form:"manufacturer"`
	Category     *string `form:"category"`
	MinPrice     *int64  `form:"min_price"`
	MaxPrice     *int64  `form:"max_price"`
	Limit        int32   `form:"limit,default=10"`
	Offset       int32   `form:"offset,default=0"`
}

type PatchProductParams struct {
	ID           int64   `json:"-"                       swaggerignore:"true"`
	ProductName  *string `json:"product_name,omitempty"  example:"Молоток с резиновой ручкой"`
	Manufacturer *string `json:"manufacturer,omitempty"  example:"Зубр"`
	Category     *string `json:"category,omitempty"      example:"Инструменты"`
	Price        *int64  `json:"price,omitempty"         example:"1499"`
	Count        *int32  `json:"count,omitempty"         example:"30"`
}

type DeleteProductParams struct {
	ID int64 `uri:"id" json:"id"`
}

type RestoreProductParams struct {
	ID int64 `uri:"id" json:"id"`
}

type GetProductParams struct {
	ID int64 `uri:"id" json:"id"`
}

type PatchProductURI struct {
	ID int64 `uri:"id" binding:"required"`
}

type CreateProductParams struct {
	ProductName  string `json:"product_name" example:"Молоток"`
	Manufacturer string `json:"manufacturer" example:"Зубр"`
	Category     string `json:"category"     example:"Инструменты"`
	Price        int64  `json:"price"        example:"1299"`
	Count        int32  `json:"count"        example:"50"`
}

type ListProductsParams struct {
	Limit  int32 `form:"limit,default=10"  json:"limit"`
	Offset int32 `form:"offset,default=0"  json:"offset"`
}
