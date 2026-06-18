// Package event описывает доменные события склада и их сериализацию.
// Пакет остаётся чистым: без зависимостей от БД, брокера и прочей инфраструктуры.
package event

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Event — доменное событие. EventName используется как топик Kafka,
// Key — как ключ партиционирования.
type Event interface {
	EventName() string
	Key() string
}

// Encode сериализует событие для отправки в брокер: topic, ключ партиционирования и JSON-payload.
func Encode(e Event) (topic, key string, payload []byte, err error) {
	payload, err = json.Marshal(e)
	if err != nil {
		return "", "", nil, fmt.Errorf("marshal event %s: %w", e.EventName(), err)
	}
	return e.EventName(), e.Key(), payload, nil
}

type ProductCreated struct {
	ID           int64  `json:"id"`
	ProductName  string `json:"product_name"`
	Manufacturer string `json:"manufacturer"`
	Category     string `json:"category"`
	Price        int64  `json:"price"`
	Count        int32  `json:"count"`
}

func (ProductCreated) EventName() string { return "product.created" }
func (e ProductCreated) Key() string     { return strconv.FormatInt(e.ID, 10) }

type ProductDeleted struct {
	ID int64 `json:"id"`
}

func (ProductDeleted) EventName() string { return "product.deleted" }
func (e ProductDeleted) Key() string     { return strconv.FormatInt(e.ID, 10) }
