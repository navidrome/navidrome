package diodes

import (
	"context"

	"code.cloudfoundry.org/go-diodes"
)

type Diode[T any] struct {
	d *diodes.Waiter
}

type Alerter = diodes.Alerter

type AlertFunc = diodes.AlertFunc

func New[T any](ctx context.Context, size int, alerter Alerter) *Diode[T] {
	return &Diode[T]{
		d: diodes.NewWaiter(diodes.NewOneToOne(size, alerter), diodes.WithWaiterContext(ctx)),
	}
}

func (d *Diode[T]) Put(data T) {
	d.d.Set(diodes.GenericDataType(&data))
}

func (d *Diode[T]) TryNext() (*T, bool) {
	data, ok := d.d.TryNext()
	if !ok {
		return nil, ok
	}
	return (*T)(data), true
}

func (d *Diode[T]) Next() *T {
	data := d.d.Next()
	return (*T)(data)
}
