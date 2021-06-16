package events

import (
	"context"

	"code.cloudfoundry.org/go-diodes"
)

type diode struct {
	d *diodes.Waiter
}

func newDiode(ctx context.Context, size int, alerter diodes.Alerter) *diode {
	return &diode{
		d: diodes.NewWaiter(diodes.NewOneToOne(size, alerter), diodes.WithWaiterContext(ctx)),
	}
}

func (d *diode) put(data message) {
	d.d.Set(diodes.GenericDataType(&data))
}

func (d *diode) tryNext() (*message, bool) {
	data, ok := d.d.TryNext()
	if !ok {
		return nil, ok
	}
	return (*message)(data), true
}

func (d *diode) next() *message {
	data := d.d.Next()
	return (*message)(data)
}
