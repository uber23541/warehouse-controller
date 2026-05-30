package event

import "context"

type Event interface {
	EventName() string
}

type Publisher interface {
	Publish(ctx context.Context, e Event) error
}

type ProductCreated struct {
	ID           int64
	ProductName  string
	Manufacturer string
	Category     string
	Price        int64
	Count        int32
}

func (ProductCreated) EventName() string { return "product.created" }

type ProductDeleted struct {
	ID int64
}

func (ProductDeleted) EventName() string { return "product.deleted" }
