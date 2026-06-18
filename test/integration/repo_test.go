//go:build integration

package integration

import (
	"context"
	"testing"

	"warehouse-controller/internal/repo"
	"warehouse-controller/internal/repo/dbmodel"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strptr(s string) *string { return &s }
func i64ptr(v int64) *int64   { return &v }

func sampleProduct() *dbmodel.Product {
	return &dbmodel.Product{
		ProductName:  "Молоток",
		Manufacturer: "Зубр",
		Category:     "Инструменты",
		Count:        10,
		Price:        1299,
	}
}

func TestRepo_CreateAndGetByID(t *testing.T) {
	ctx := context.Background()
	r := repo.NewProductRepo(startPostgres(t))

	id, err := r.Create(ctx, sampleProduct())
	require.NoError(t, err)
	require.Positive(t, id)

	got, err := r.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "Молоток", got.ProductName)
	assert.Equal(t, int64(1299), got.Price)
	assert.Equal(t, int32(10), got.Count)
}

func TestRepo_DeleteHidesProduct(t *testing.T) {
	ctx := context.Background()
	r := repo.NewProductRepo(startPostgres(t))

	id, err := r.Create(ctx, sampleProduct())
	require.NoError(t, err)

	require.NoError(t, r.Delete(ctx, id))

	_, err = r.GetByID(ctx, id)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestRepo_RestoreProduct(t *testing.T) {
	ctx := context.Background()
	r := repo.NewProductRepo(startPostgres(t))

	id, err := r.Create(ctx, sampleProduct())
	require.NoError(t, err)
	require.NoError(t, r.Delete(ctx, id))

	restored, err := r.Restore(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, id, restored.ID)

	got, err := r.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestRepo_PatchPartial(t *testing.T) {
	ctx := context.Background()
	r := repo.NewProductRepo(startPostgres(t))

	id, err := r.Create(ctx, sampleProduct())
	require.NoError(t, err)

	// Меняем только цену — остальные поля должны сохраниться (COALESCE).
	patched, err := r.Patch(ctx, id, dbmodel.ProductPatch{Price: i64ptr(2000)})
	require.NoError(t, err)
	assert.Equal(t, int64(2000), patched.Price)
	assert.Equal(t, "Молоток", patched.ProductName)
	assert.Equal(t, "Зубр", patched.Manufacturer)
}

func TestRepo_Search(t *testing.T) {
	ctx := context.Background()
	r := repo.NewProductRepo(startPostgres(t))

	_, err := r.Create(ctx, &dbmodel.Product{ProductName: "Молоток", Manufacturer: "Зубр", Category: "Инструменты", Count: 1, Price: 100})
	require.NoError(t, err)
	_, err = r.Create(ctx, &dbmodel.Product{ProductName: "Дрель", Manufacturer: "Bosch", Category: "Электро", Count: 1, Price: 5000})
	require.NoError(t, err)
	_, err = r.Create(ctx, &dbmodel.Product{ProductName: "Молоток малый", Manufacturer: "Зубр", Category: "Инструменты", Count: 1, Price: 300})
	require.NoError(t, err)

	byName, err := r.Search(ctx, dbmodel.ProductFilter{ProductName: strptr("Молот"), Limit: 10})
	require.NoError(t, err)
	assert.Len(t, byName, 2)

	byManufacturer, err := r.Search(ctx, dbmodel.ProductFilter{Manufacturer: strptr("Bosch"), Limit: 10})
	require.NoError(t, err)
	require.Len(t, byManufacturer, 1)
	assert.Equal(t, "Дрель", byManufacturer[0].ProductName)

	byPrice, err := r.Search(ctx, dbmodel.ProductFilter{MinPrice: i64ptr(200), MaxPrice: i64ptr(1000), Limit: 10})
	require.NoError(t, err)
	require.Len(t, byPrice, 1)
	assert.Equal(t, "Молоток малый", byPrice[0].ProductName)
}

func TestRepo_ListOrderAndPagination(t *testing.T) {
	ctx := context.Background()
	r := repo.NewProductRepo(startPostgres(t))

	var ids []int64
	for range 3 {
		id, err := r.Create(ctx, sampleProduct())
		require.NoError(t, err)
		ids = append(ids, id)
	}

	all, err := r.List(ctx, 10, 0)
	require.NoError(t, err)
	require.Len(t, all, 3)
	// ORDER BY id DESC — последний созданный первым.
	assert.Equal(t, ids[2], all[0].ID)

	page, err := r.List(ctx, 1, 1)
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Equal(t, ids[1], page[0].ID)
}
