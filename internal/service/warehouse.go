package service

import (
	"context"
	"warehouse-controller/internal/domain"
	"warehouse-controller/internal/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type WarehouseService struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	logger  *zap.Logger
}

func NewWarehouseService(pool *pgxpool.Pool, queries *sqlc.Queries, logger *zap.Logger) *WarehouseService {
	return &WarehouseService{pool: pool, queries: queries, logger: logger}
}

func (s *WarehouseService) CreateProduct(ctx context.Context, req domain.CreateProductParams) (int64, error) {
	id, err := s.queries.CreateProduct(ctx, sqlc.CreateProductParams{
		ProductName:  req.ProductName,
		Manufacturer: req.Manufacturer,
		Category:     req.Category,
		Count:        req.Count,
		Price:        req.Price,
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *WarehouseService) GetProductByID(ctx context.Context, req domain.GetProductParams) (*domain.Product, error) {
	product, err := s.queries.GetProductByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &domain.Product{
		ID:           product.ID,
		ProductName:  product.ProductName,
		Manufacturer: product.Manufacturer,
		Category:     product.Category,
		Price:        product.Price,
		Count:        product.Count,
	}, nil
}

func (s *WarehouseService) DeleteProduct(ctx context.Context, req domain.DeleteProductParams) error {
	return s.queries.DeleteProduct(ctx, req.ID)
}

func (s *WarehouseService) RestoreProduct(ctx context.Context, req domain.RestoreProductParams) (*domain.Product, error) {
	product, err := s.queries.RestoreProduct(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &domain.Product{
		ID:           product.ID,
		ProductName:  product.ProductName,
		Manufacturer: product.Manufacturer,
		Category:     product.Category,
		Price:        product.Price,
		Count:        product.Count,
	}, nil
}

func (s *WarehouseService) SearchProducts(ctx context.Context, req domain.SearchProductsParams) ([]domain.Product, error) {
	products, err := s.queries.SearchProducts(ctx, sqlc.SearchProductsParams{
		ProductName:  toPgText(req.ProductName),
		Manufacturer: toPgText(req.Manufacturer),
		Category:     toPgText(req.Category),
		MinPrice:     toPgInt8(req.MinPrice),
		MaxPrice:     toPgInt8(req.MaxPrice),
		Limit:        req.Limit,
		Offset:       req.Offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]domain.Product, len(products))
	for i, product := range products {
		result[i] = domain.Product{
			ID:           product.ID,
			ProductName:  product.ProductName,
			Manufacturer: product.Manufacturer,
			Category:     product.Category,
			Price:        product.Price,
			Count:        product.Count,
		}
	}
	return result, nil
}

func (s *WarehouseService) PatchProduct(ctx context.Context, req domain.PatchProductParams) (*domain.Product, error) {
	product, err := s.queries.PatchProduct(ctx, sqlc.PatchProductParams{
		ProductName:  toPgText(req.ProductName),
		Manufacturer: toPgText(req.Manufacturer),
		Category:     toPgText(req.Category),
		Price:        toPgInt8(req.Price),
		Count:        toPgInt4(req.Count),
		ID:           req.ID,
	})
	if err != nil {
		return nil, err
	}
	return &domain.Product{
		ID:           product.ID,
		ProductName:  product.ProductName,
		Manufacturer: product.Manufacturer,
		Category:     product.Category,
		Price:        product.Price,
		Count:        product.Count,
	}, nil
}

func (s *WarehouseService) ListProducts(ctx context.Context, req domain.ListProductsParams) ([]domain.Product, error) {
	products, err := s.queries.ListProducts(ctx, sqlc.ListProductsParams{
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]domain.Product, len(products))
	for i, product := range products {
		result[i] = domain.Product{
			ID:           product.ID,
			ProductName:  product.ProductName,
			Manufacturer: product.Manufacturer,
			Category:     product.Category,
			Price:        product.Price,
			Count:        product.Count,
		}
	}
	return result, nil
}

func toPgText(v *string) pgtype.Text {
	if v == nil {
		return pgtype.Text{Valid: false}
	}

	return pgtype.Text{
		String: *v,
		Valid:  true,
	}
}

func toPgInt8(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{Valid: false}
	}

	return pgtype.Int8{
		Int64: *v,
		Valid: true,
	}
}

func toPgInt4(v *int32) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{Valid: false}
	}

	return pgtype.Int4{
		Int32: *v,
		Valid: true,
	}
}
