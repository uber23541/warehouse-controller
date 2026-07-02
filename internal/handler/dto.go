package handler

import (
	"warehouse-controller/internal/domain"
	"warehouse-controller/internal/service"
)

type ErrorResponse struct {
	Error string `json:"error" example:"invalid request body"`
}

type ProductIDURI struct {
	ID int64 `uri:"id" binding:"required"`
}

type CreateProductRequest struct {
	ProductName  string `json:"product_name" example:"Молоток"`
	Manufacturer string `json:"manufacturer" example:"Зубр"`
	Category     string `json:"category"     example:"Инструменты"`
	Price        int64  `json:"price"        example:"1299"`
	Count        int32  `json:"count"        example:"50"`
}

func (r CreateProductRequest) toDomain() domain.CreateProductParams {
	return domain.CreateProductParams{
		ProductName:  r.ProductName,
		Manufacturer: r.Manufacturer,
		Category:     r.Category,
		Price:        r.Price,
		Count:        r.Count,
	}
}

type SearchProductsRequest struct {
	ProductName  *string `form:"product_name"`
	Manufacturer *string `form:"manufacturer"`
	Category     *string `form:"category"`
	MinPrice     *int64  `form:"min_price"`
	MaxPrice     *int64  `form:"max_price"`
	Limit        int32   `form:"limit,default=10"`
	Offset       int32   `form:"offset,default=0"`
}

func (r SearchProductsRequest) toDomain() domain.SearchProductsParams {
	return domain.SearchProductsParams{
		ProductName:  r.ProductName,
		Manufacturer: r.Manufacturer,
		Category:     r.Category,
		MinPrice:     r.MinPrice,
		MaxPrice:     r.MaxPrice,
		Limit:        r.Limit,
		Offset:       r.Offset,
	}
}

type PatchProductRequest struct {
	ProductName  *string `json:"product_name,omitempty" example:"Молоток с резиновой ручкой"`
	Manufacturer *string `json:"manufacturer,omitempty" example:"Зубр"`
	Category     *string `json:"category,omitempty"     example:"Инструменты"`
	Price        *int64  `json:"price,omitempty"        example:"1499"`
	Count        *int32  `json:"count,omitempty"        example:"30"`
}

func (r PatchProductRequest) toDomain(id int64) domain.PatchProductParams {
	return domain.PatchProductParams{
		ID:           id,
		ProductName:  r.ProductName,
		Manufacturer: r.Manufacturer,
		Category:     r.Category,
		Price:        r.Price,
		Count:        r.Count,
	}
}

type ListProductsRequest struct {
	Limit  int32 `form:"limit,default=10"`
	Offset int32 `form:"offset,default=0"`
}

func (r ListProductsRequest) toDomain() domain.ListProductsParams {
	return domain.ListProductsParams{
		Limit:  r.Limit,
		Offset: r.Offset,
	}
}

type RefreshRequest struct {
	Refresh string `json:"refresh" binding:"required"`
}

type CreateProductResponse struct {
	ID int64 `json:"id" example:"42"`
}

type ProductResponse struct {
	ID           int64  `json:"id"           example:"42"`
	ProductName  string `json:"product_name" example:"Молоток"`
	Manufacturer string `json:"manufacturer" example:"Зубр"`
	Category     string `json:"category"     example:"Инструменты"`
	Price        int64  `json:"price"        example:"1299"`
	Count        int32  `json:"count"        example:"50"`
}

func newProductResponse(p *domain.Product) ProductResponse {
	return ProductResponse{
		ID:           p.ID,
		ProductName:  p.ProductName,
		Manufacturer: p.Manufacturer,
		Category:     p.Category,
		Price:        p.Price,
		Count:        p.Count,
	}
}

func newProductResponses(products []domain.Product) []ProductResponse {
	result := make([]ProductResponse, len(products))
	for i := range products {
		result[i] = newProductResponse(&products[i])
	}
	return result
}

type TokenPairResponse struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

func newTokenPairResponse(p service.TokenPair) TokenPairResponse {
	return TokenPairResponse{Access: p.Access, Refresh: p.Refresh}
}
