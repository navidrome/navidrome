package nativeapi

import (
	"context"

	"github.com/deluan/rest"
)

// lazyRepo builds the repository per request. Removed once all repositories are built once.
type lazyRepo[T any] struct {
	new func(context.Context) rest.Repository[T]
}

func (l lazyRepo[T]) Count(ctx context.Context, o ...rest.QueryOptions) (int64, error) {
	return l.new(ctx).Count(ctx, o...)
}

func (l lazyRepo[T]) Read(ctx context.Context, id string) (*T, error) {
	return l.new(ctx).Read(ctx, id)
}

func (l lazyRepo[T]) ReadAll(ctx context.Context, o ...rest.QueryOptions) ([]T, error) {
	return l.new(ctx).ReadAll(ctx, o...)
}

type lazyPersistable[T any] struct {
	lazyRepo[T]
}

func (l lazyPersistable[T]) Save(ctx context.Context, e *T) (string, error) {
	return l.new(ctx).(rest.Persistable[T]).Save(ctx, e)
}

func (l lazyPersistable[T]) Update(ctx context.Context, id string, e T, fields ...string) error {
	return l.new(ctx).(rest.Persistable[T]).Update(ctx, id, e, fields...)
}

func (l lazyPersistable[T]) Delete(ctx context.Context, ids ...string) error {
	return l.new(ctx).(rest.Persistable[T]).Delete(ctx, ids...)
}

func lazyRW[T any](new func(context.Context) rest.Repository[T]) rest.Repository[T] {
	return lazyPersistable[T]{lazyRepo[T]{new: new}}
}
