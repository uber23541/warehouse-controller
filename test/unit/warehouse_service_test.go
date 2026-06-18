package unit

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"warehouse-controller/internal/platform/cache"
	productcache "warehouse-controller/internal/platform/cache/product"
	"warehouse-controller/internal/domain"
	cachemock "warehouse-controller/internal/mocks/cache"
	postgresmock "warehouse-controller/internal/mocks/postgres"
	repomock "warehouse-controller/internal/mocks/repo"
	servicemock "warehouse-controller/internal/mocks/service"
	"warehouse-controller/internal/outbox"
	"warehouse-controller/internal/repo/dbmodel"
	"warehouse-controller/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newWarehouseService(t *testing.T) (*service.WarehouseService, *repomock.MockProductRepository, *cachemock.MockCache, *servicemock.MockOutboxStore) {
	t.Helper()
	r := repomock.NewMockProductRepository(t)
	c := cachemock.NewMockCache(t)
	ob := servicemock.NewMockOutboxStore(t)
	txm := postgresmock.NewMockTransactor(t)
	// Транзакция в тестах прозрачна: просто выполняем переданную функцию.
	txm.EXPECT().WithinTx(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).Maybe()
	svc := service.NewWarehouseService(r, c, txm, ob, zap.NewNop())
	return svc, r, c, ob
}

// outboxTopic матчит запись outbox по имени топика.
func outboxTopic(name string) func(outbox.Record) bool {
	return func(rec outbox.Record) bool { return rec.Topic == name }
}

func marshalProduct(t *testing.T, p *domain.Product) []byte {
	t.Helper()
	data, err := json.Marshal(productcache.FromDomain(p))
	require.NoError(t, err)
	return data
}

func marshalProducts(t *testing.T, ps []domain.Product) []byte {
	t.Helper()
	data, err := json.Marshal(productcache.FromDomainSlice(ps))
	require.NoError(t, err)
	return data
}

func TestWarehouseService_CreateProduct(t *testing.T) {
	createErr := errors.New("insert failed")
	outboxErr := errors.New("outbox down")

	tests := []struct {
		name    string
		req     domain.CreateProductParams
		setup   func(r *repomock.MockProductRepository, ob *servicemock.MockOutboxStore)
		wantID  int64
		wantErr error
	}{
		{
			name: "ok: создаёт и сохраняет событие в outbox",
			req:  domain.CreateProductParams{ProductName: "Молоток", Manufacturer: "Зубр", Category: "Инструменты", Count: 5, Price: 1299},
			setup: func(r *repomock.MockProductRepository, ob *servicemock.MockOutboxStore) {
				r.EXPECT().Create(context.Background(), &dbmodel.Product{ProductName: "Молоток", Manufacturer: "Зубр", Category: "Инструменты", Count: 5, Price: 1299}).
					Return(int64(42), nil).Once()
				ob.EXPECT().Save(context.Background(), mock.MatchedBy(outboxTopic("product.created"))).Return(nil).Once()
			},
			wantID: 42,
		},
		{
			name: "repo error: событие не пишется",
			req:  domain.CreateProductParams{},
			setup: func(r *repomock.MockProductRepository, _ *servicemock.MockOutboxStore) {
				r.EXPECT().Create(context.Background(), mock.Anything).Return(int64(0), createErr).Once()
			},
			wantErr: createErr,
		},
		{
			name: "ошибка outbox откатывает операцию",
			req:  domain.CreateProductParams{ProductName: "X"},
			setup: func(r *repomock.MockProductRepository, ob *servicemock.MockOutboxStore) {
				r.EXPECT().Create(context.Background(), mock.Anything).Return(int64(7), nil).Once()
				ob.EXPECT().Save(context.Background(), mock.Anything).Return(outboxErr).Once()
			},
			wantErr: outboxErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, r, _, ob := newWarehouseService(t)
			tt.setup(r, ob)

			id, err := svc.CreateProduct(context.Background(), tt.req)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, id)
		})
	}
}

func TestWarehouseService_GetProductByID(t *testing.T) {
	repoErr := errors.New("db error")

	tests := []struct {
		name     string
		id       int64
		setup    func(r *repomock.MockProductRepository, c *cachemock.MockCache)
		wantName string // ожидаемое имя продукта при успехе
		wantNil  bool   // продукт nil без ошибки
		wantErr  error
	}{
		{
			name: "cache hit: репозиторий не вызывается",
			id:   1,
			setup: func(_ *repomock.MockProductRepository, c *cachemock.MockCache) {
				want := &domain.Product{ID: 1, ProductName: "Молоток", Price: 100, Count: 3}
				c.EXPECT().Get(context.Background(), "product:1").Return(marshalProduct(t, want), nil).Once()
			},
			wantName: "Молоток",
		},
		{
			name: "cache miss: читает из репозитория и кэширует",
			id:   2,
			setup: func(r *repomock.MockProductRepository, c *cachemock.MockCache) {
				c.EXPECT().Get(context.Background(), "product:2").Return(nil, cache.ErrNotFound).Times(2)
				r.EXPECT().GetByID(context.Background(), int64(2)).Return(&dbmodel.Product{ID: 2, ProductName: "Дрель", Price: 5000, Count: 1}, nil).Once()
				c.EXPECT().Set(context.Background(), "product:2", mock.Anything, mock.Anything).Return(nil).Once()
			},
			wantName: "Дрель",
		},
		{
			name: "не найдено: возвращает nil без ошибки",
			id:   3,
			setup: func(r *repomock.MockProductRepository, c *cachemock.MockCache) {
				c.EXPECT().Get(context.Background(), "product:3").Return(nil, cache.ErrNotFound).Times(2)
				r.EXPECT().GetByID(context.Background(), int64(3)).Return(nil, nil).Once()
			},
			wantNil: true,
		},
		{
			name: "ошибка репозитория",
			id:   4,
			setup: func(r *repomock.MockProductRepository, c *cachemock.MockCache) {
				c.EXPECT().Get(context.Background(), "product:4").Return(nil, cache.ErrNotFound).Times(2)
				r.EXPECT().GetByID(context.Background(), int64(4)).Return(nil, repoErr).Once()
			},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, r, c, _ := newWarehouseService(t)
			tt.setup(r, c)

			got, err := svc.GetProductByID(context.Background(), domain.GetProductParams{ID: tt.id})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.wantName, got.ProductName)
		})
	}
}

// При конкурентных промахах по одному ключу к репозиторию идёт ровно один запрос.
func TestWarehouseService_GetProductByID_SingleFlight(t *testing.T) {
	svc, r, c, _ := newWarehouseService(t)
	ctx := context.Background()

	c.EXPECT().Get(ctx, "product:9").Return(nil, cache.ErrNotFound).Maybe()
	c.EXPECT().Set(ctx, "product:9", mock.Anything, mock.Anything).Return(nil).Maybe()

	var calls atomic.Int32
	release := make(chan struct{})
	r.EXPECT().GetByID(ctx, int64(9)).RunAndReturn(func(_ context.Context, _ int64) (*dbmodel.Product, error) {
		calls.Add(1)
		<-release
		return &dbmodel.Product{ID: 9, ProductName: "Пила"}, nil
	}).Once()

	const n = 8
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			_, err := svc.GetProductByID(ctx, domain.GetProductParams{ID: 9})
			assert.NoError(t, err)
		}()
	}

	assert.Eventually(t, func() bool { return calls.Load() >= 1 }, 2*time.Second, time.Millisecond)
	close(release)
	wg.Wait()

	assert.Equal(t, int32(1), calls.Load())
}

func TestWarehouseService_SearchProducts_CacheHit(t *testing.T) {
	svc, _, c, _ := newWarehouseService(t)
	ctx := context.Background()

	want := []domain.Product{{ID: 1, ProductName: "A"}, {ID: 2, ProductName: "B"}}
	c.EXPECT().Get(ctx, "products:search:10|0").Return(marshalProducts(t, want), nil).Once()

	got, err := svc.SearchProducts(ctx, domain.SearchProductsParams{Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// Формирование ключа кэша из комбинаций фильтров (косвенно проверяет
// hashSearchParams): порядок частей и пропуск nil-полей.
func TestWarehouseService_SearchProducts_CacheKey(t *testing.T) {
	ptr := func(s string) *string { return &s }
	i64 := func(v int64) *int64 { return &v }

	tests := []struct {
		name    string
		params  domain.SearchProductsParams
		wantKey string
	}{
		{"only pagination", domain.SearchProductsParams{Limit: 10, Offset: 0}, "products:search:10|0"},
		{"name", domain.SearchProductsParams{ProductName: ptr("foo"), Limit: 5, Offset: 2}, "products:search:foo|5|2"},
		{"manufacturer", domain.SearchProductsParams{Manufacturer: ptr("bar"), Limit: 10, Offset: 0}, "products:search:bar|10|0"},
		{"category", domain.SearchProductsParams{Category: ptr("cat"), Limit: 10, Offset: 0}, "products:search:cat|10|0"},
		{"min price", domain.SearchProductsParams{MinPrice: i64(100), Limit: 10, Offset: 0}, "products:search:100|10|0"},
		{"max price", domain.SearchProductsParams{MaxPrice: i64(500), Limit: 10, Offset: 0}, "products:search:500|10|0"},
		{"all fields", domain.SearchProductsParams{ProductName: ptr("a"), Manufacturer: ptr("b"), Category: ptr("c"), MinPrice: i64(1), MaxPrice: i64(2), Limit: 7, Offset: 3}, "products:search:a|b|c|1|2|7|3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, r, c, _ := newWarehouseService(t)
			ctx := context.Background()

			c.EXPECT().Get(ctx, tt.wantKey).Return(nil, cache.ErrNotFound).Times(2)
			r.EXPECT().Search(ctx, mock.Anything).Return(nil, nil).Once()
			c.EXPECT().Set(ctx, tt.wantKey, mock.Anything, mock.Anything).Return(nil).Once()

			_, err := svc.SearchProducts(ctx, tt.params)
			require.NoError(t, err)
		})
	}
}

func TestWarehouseService_SearchProducts_CacheMiss(t *testing.T) {
	svc, r, c, _ := newWarehouseService(t)
	ctx := context.Background()

	name := "Молот"
	key := "products:search:Молот|20|5"
	c.EXPECT().Get(ctx, key).Return(nil, cache.ErrNotFound).Times(2)
	r.EXPECT().Search(ctx, mock.MatchedBy(func(f dbmodel.ProductFilter) bool {
		return f.ProductName != nil && *f.ProductName == "Молот" && f.Limit == 20 && f.Offset == 5
	})).Return([]dbmodel.Product{{ID: 1, ProductName: "Молоток"}}, nil).Once()
	c.EXPECT().Set(ctx, key, mock.Anything, mock.Anything).Return(nil).Once()

	got, err := svc.SearchProducts(ctx, domain.SearchProductsParams{ProductName: &name, Limit: 20, Offset: 5})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Молоток", got[0].ProductName)
}

func TestWarehouseService_ListProducts_CacheMiss(t *testing.T) {
	svc, r, c, _ := newWarehouseService(t)
	ctx := context.Background()

	key := "products:list:10:0"
	c.EXPECT().Get(ctx, key).Return(nil, cache.ErrNotFound).Times(2)
	r.EXPECT().List(ctx, int32(10), int32(0)).Return([]dbmodel.Product{{ID: 1}}, nil).Once()
	c.EXPECT().Set(ctx, key, mock.Anything, mock.Anything).Return(nil).Once()

	got, err := svc.ListProducts(ctx, domain.ListProductsParams{Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestWarehouseService_DeleteProduct(t *testing.T) {
	deleteErr := errors.New("delete failed")

	tests := []struct {
		name    string
		setup   func(r *repomock.MockProductRepository, ob *servicemock.MockOutboxStore)
		wantErr error
	}{
		{
			name: "ok: удаляет и сохраняет событие в outbox",
			setup: func(r *repomock.MockProductRepository, ob *servicemock.MockOutboxStore) {
				r.EXPECT().Delete(context.Background(), int64(5)).Return(nil).Once()
				ob.EXPECT().Save(context.Background(), mock.MatchedBy(outboxTopic("product.deleted"))).Return(nil).Once()
			},
		},
		{
			name: "repo error",
			setup: func(r *repomock.MockProductRepository, _ *servicemock.MockOutboxStore) {
				r.EXPECT().Delete(context.Background(), int64(5)).Return(deleteErr).Once()
			},
			wantErr: deleteErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, r, _, ob := newWarehouseService(t)
			tt.setup(r, ob)

			err := svc.DeleteProduct(context.Background(), domain.DeleteProductParams{ID: 5})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestWarehouseService_RestoreProduct_NotFound(t *testing.T) {
	svc, r, _, _ := newWarehouseService(t)
	ctx := context.Background()

	r.EXPECT().Restore(ctx, int64(8)).Return(nil, nil).Once()

	got, err := svc.RestoreProduct(ctx, domain.RestoreProductParams{ID: 8})
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestWarehouseService_PatchProduct(t *testing.T) {
	newName := "Молоток PRO"

	tests := []struct {
		name     string
		params   domain.PatchProductParams
		setup    func(r *repomock.MockProductRepository)
		wantNil  bool
		wantName string
	}{
		{
			name:   "ok: обновляет имя",
			params: domain.PatchProductParams{ID: 12, ProductName: &newName},
			setup: func(r *repomock.MockProductRepository) {
				r.EXPECT().Patch(context.Background(), int64(12), mock.MatchedBy(func(p dbmodel.ProductPatch) bool {
					return p.ProductName != nil && *p.ProductName == "Молоток PRO"
				})).Return(&dbmodel.Product{ID: 12, ProductName: "Молоток PRO"}, nil).Once()
			},
			wantName: "Молоток PRO",
		},
		{
			name:   "не найдено: возвращает nil без ошибки",
			params: domain.PatchProductParams{ID: 11},
			setup: func(r *repomock.MockProductRepository) {
				r.EXPECT().Patch(context.Background(), int64(11), mock.Anything).Return(nil, nil).Once()
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, r, _, _ := newWarehouseService(t)
			tt.setup(r)

			got, err := svc.PatchProduct(context.Background(), tt.params)

			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.wantName, got.ProductName)
		})
	}
}
