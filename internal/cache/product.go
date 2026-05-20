package cache

import "warehouse-controller/internal/domain"

type Product struct {
	ID           int64  `json:"id"`
	ProductName  string `json:"product_name"`
	Manufacturer string `json:"manufacturer"`
	Category     string `json:"category"`
	Price        int64  `json:"price"`
	Count        int32  `json:"count"`
}

func FromDomain(p *domain.Product) *Product {
	if p == nil {
		return nil
	}
	return &Product{
		ID:           p.ID,
		ProductName:  p.ProductName,
		Manufacturer: p.Manufacturer,
		Category:     p.Category,
		Price:        p.Price,
		Count:        p.Count,
	}
}

func FromDomainSlice(products []domain.Product) []Product {
	result := make([]Product, len(products))
	for i := range products {
		result[i] = *FromDomain(&products[i])
	}
	return result
}

func (p *Product) ToDomain() *domain.Product {
	if p == nil {
		return nil
	}
	return &domain.Product{
		ID:           p.ID,
		ProductName:  p.ProductName,
		Manufacturer: p.Manufacturer,
		Category:     p.Category,
		Price:        p.Price,
		Count:        p.Count,
	}
}

func ToDomainSlice(products []Product) []domain.Product {
	result := make([]domain.Product, len(products))
	for i := range products {
		result[i] = *products[i].ToDomain()
	}
	return result
}
